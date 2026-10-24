const apiUrl = Cypress.env('apiUrl');

// Subscriber e-mail links in the table, one per row (each carries data-id).
const emails = () => cy.get('.p-datatable-tbody [data-cy=email]');

// Opens a ListSelector's dropdown (within `scope`) and picks the first list
// not already selected.
const pickFirstList = (scope) => {
  scope().find('.list-selector .p-autocomplete-dropdown').click();
  cy.get('.p-autocomplete-overlay .p-autocomplete-option').first().click();
  cy.get('.p-autocomplete-overlay').should('not.exist');
};

// The subscription statuses of a subscriber's lists, in API order.
const listStatuses = (id) => cy.request(`${apiUrl}/api/subscribers/${id}`)
  .then((resp) => resp.body.data.lists.map((l) => l.subscription_status));

// Opens the row menu of the table row at `index` and clicks `label`.
const rowAction = (index, label) => {
  cy.rows().eq(index).find('[data-cy=row-menu]').click();
  cy.get('.p-menu .p-menu-item').contains(label).click();
};

// Fills and saves the add / edit subscriber dialog.
const fillForm = ({
  email, name, status, numLists, attribs,
}) => {
  cy.dialog().find('input[name=email]').clear().type(email);
  cy.dialog().find('input[name=name]').clear().type(name);
  if (status) {
    cy.pvSelect('[name=status]', status === 'blocklisted' ? 'Blocklisted' : 'Enabled');
  }
  Cypress._.times(numLists, () => pickFirstList(cy.dialog));
  cy.dialog().find('[data-cy=tab-attribs]').click();
  cy.dialog().find('textarea[name=attribs]').clear().type(attribs, { parseSpecialCharSequences: false, delay: 0 });
  cy.dialog().find('[data-cy=btn-save]').click();
  cy.get('.subscriber-modal').should('not.exist');
};

// Asserts a table row (the <tr> holding the subscriber link) shows the
// subscriber's e-mail, name, blocklisted state and active list count, and
// that the API reports the expected status on each of its lists.
const checkRow = ($tr, {
  id, email, name, status, numLists,
}) => {
  cy.wrap($tr).find('[data-cy=email]').should('contain', email.toLowerCase());
  cy.wrap($tr).find('[data-cy=name]').should('contain', name);
  cy.wrap($tr).find('[data-cy=blocklisted]').should(status === 'blocklisted' ? 'exist' : 'not.exist');

  // The Lists column leaves out unsubscribed lists.
  cy.wrap($tr).find('[data-cy=lists] .ln-attr').should('have.length', status === 'blocklisted' ? 0 : numLists);

  // Lists on an enabled subscriber are 'unconfirmed' and on a blocklisted one, 'unsubscribed'.
  listStatuses(id).then((statuses) => {
    expect(statuses).to.have.lengthOf(numLists);
    statuses.forEach((s) => expect(s).to.equal(status === 'blocklisted' ? 'unsubscribed' : 'unconfirmed'));
  });
};

describe('Subscribers', () => {
  it('Opens subscribers page', () => {
    cy.resetDB();
    cy.loginAndVisit('/admin/subscribers');
  });

  it('Counts subscribers', () => {
    emails().should('have.length', 2);
  });

  it('Searches subscribers', () => {
    const cases = [
      { value: 'john{enter}', count: 1, contains: 'john@example.com' },
      { value: 'anon{enter}', count: 1, contains: 'anon@example.com' },
      { value: '{enter}', count: 2, contains: null },
    ];

    cases.forEach((c) => {
      cy.get('[data-cy=search]').clear().type(c.value);
      emails().should('have.length', c.count);
      if (c.contains) {
        emails().contains(c.contains);
      }
    });
  });

  it('Exports subscribers', () => {
    const cases = [
      {
        listIDs: [], ids: [], query: '', length: 3,
      },
      {
        listIDs: [], ids: [], query: "name ILIKE '%anon%'", length: 2,
      },
      {
        listIDs: [], ids: [], query: "name like 'nope'", length: 1,
      },
    ];

    // listIDs[] and ids[] are unused for now as Cypress doesn't support encoding of arrays in `qs`.
    cases.forEach((c) => {
      cy.request({ url: `${apiUrl}/api/subscribers/export`, qs: { query: c.query, list_id: c.listIDs, id: c.ids } }).then((resp) => {
        cy.expect(resp.body.trim().split('\n')).to.have.lengthOf(c.length);
      });
    });
  });

  it('Advanced searches subscribers', () => {
    // The SQL box opens from the segment builder's "Edit as SQL".
    cy.get('[data-cy=btn-edit-sql]').click();

    const cases = [
      { value: 'subscribers.attribs->>\'city\'=\'Bengaluru\'', count: 2 },
      { value: 'subscribers.attribs->>\'city\'=\'Bengaluru\' AND id=1', count: 1 },
      { value: '(subscribers.attribs->>\'good\')::BOOLEAN = true AND name like \'Anon%\'', count: 1 },
    ];

    cases.forEach((c) => {
      cy.get('[data-cy=query]').clear().type(c.value);
      cy.get('[data-cy=btn-query]').click();
      emails().should('have.length', c.count);
    });

    // While the advanced query is open, the same button resets it.
    cy.get('[data-cy=btn-advanced-search]').click();
    cy.get('[data-cy=query]').should('not.exist');
    emails().should('have.length', 2);
  });

  it('Does bulk subscriber list add and remove', () => {
    const cases = [
      // radio: action to perform, rows: table rows to select and perform on: [expected statuses of those rows' lists after the action]
      { radio: 'check-list-add', lists: [0, 1], rows: { 0: ['confirmed', 'confirmed'] } },
      { radio: 'check-list-unsubscribe', lists: [0, 1], rows: { 0: ['unsubscribed', 'unsubscribed'], 1: ['unsubscribed'] } },
      { radio: 'check-list-remove', lists: [0, 1], rows: { 1: [] } },
      { radio: 'check-list-add', lists: [0, 1], rows: { 0: ['unsubscribed', 'unsubscribed'], 1: ['unconfirmed', 'unconfirmed'] } },
      { radio: 'check-list-remove', lists: [0], rows: { 0: ['unsubscribed'] } },
      { radio: 'check-list-add', lists: [0], rows: { 0: ['unconfirmed', 'unsubscribed'] } },
    ];

    cy.intercept('PUT', '**/api/subscribers/lists').as('manageLists');
    cy.intercept({ method: 'GET', pathname: '/api/subscribers' }).as('getSubs');

    cases.forEach((c, n) => {
      const rows = Object.keys(c.rows).map(Number);

      // Select the given subscribers in the table.
      rows.forEach((r) => {
        cy.rows().eq(r).find('.p-checkbox input').check({ force: true });
      });

      // Open the 'manage lists' dialog from the bulk bar.
      cy.get('[data-cy=btn-manage-lists]').click();

      // Pick the lists in the dialog.
      c.lists.forEach(() => pickFirstList(cy.dialog));

      // Select the action.
      cy.dialog().find(`[data-cy=${c.radio}] input`).check({ force: true });

      // For the first test, check the optin preconfirm box.
      if (n === 0) {
        cy.dialog().find('[data-cy=preconfirm] input').check({ force: true });
      }

      // Save, then wait for the change and the table refresh (which also
      // clears the selection).
      cy.dialog().find('[data-cy=btn-save]').click();
      cy.wait('@manageLists');
      cy.wait('@getSubs');

      // Check the status of the lists on each subscriber. The table only
      // shows active lists, so read every subscription's status from the API.
      rows.forEach((r) => {
        emails().eq(r).invoke('attr', 'data-id').then((id) => {
          listStatuses(id).should('deep.equal', c.rows[r]);
        });
      });
    });
  });

  it('Resets subscribers page', () => {
    cy.resetDB();
    cy.loginAndVisit('/admin/subscribers');
  });

  it('Edits subscribers', () => {
    const status = ['enabled', 'blocklisted'];
    const json = '{"string": "hello", "ints": [1,2,3], "null": null, "sub": {"bool": true}}';

    // Collect values being edited on each sub to confirm the changes in the next step
    // index by their ID shown in the dialog.
    const rows = {};

    emails().should('have.length', 2);
    status.forEach((st, n) => {
      const email = `email-${n}@EMAIL.com`;
      const name = `name-${n}`;

      // Open the edit dialog from the row menu.
      rowAction(n, 'Edit');

      // Get the ID from the header and proceed to fill the form.
      cy.dialog().find('[data-cy=id]').then(($el) => {
        const id = parseInt($el.text().trim(), 10);
        rows[id] = { email, name, status: st };

        // Each seeded subscriber is on one list; add the other one.
        fillForm({
          email, name, status: st === 'blocklisted' ? st : null, numLists: 1, attribs: json,
        });
      });
    });

    // Confirm the edits on the table.
    cy.wait(500);
    emails().should('have.length', 2).each(($a) => {
      const id = parseInt($a.attr('data-id'), 10);
      checkRow($a.closest('tr'), { id, ...rows[id], numLists: 2 });
    });
  });

  it('Deletes subscribers', () => {
    // Delete all visible subscribers, one at a time from the row menu.
    emails().each(() => {
      rowAction(0, 'Delete');
      cy.confirmAction();
      cy.get('.ln-confirm').should('not.exist');
    });

    // Confirm deletion.
    cy.get('.p-datatable-empty-message');
    emails().should('not.exist');
  });

  it('Creates new subscribers', () => {
    const statuses = ['enabled', 'blocklisted'];
    const lists = [[1], [2], [1, 2]];
    const json = '{"string": "hello", "ints": [1,2,3], "null": null, "sub": {"bool": true}}';

    // Cycle through each status and each list ID combination and create subscribers.
    Cypress._.times(6, (n) => {
      const email = `email-${n}@EMAIL.com`;
      const name = `name-${n}`;
      const status = statuses[(n + 1) % statuses.length];
      const list = lists[(n + 1) % lists.length];

      cy.get('[data-cy=btn-new]').click();
      fillForm({
        email, name, status, numLists: list.length, attribs: json,
      });

      // Confirm the addition by inspecting the newly created subscriber,
      // which is always the first row in the table.
      emails().first().should('contain', email.toLowerCase()).then(($a) => {
        const id = parseInt($a.attr('data-id'), 10);
        checkRow($a.closest('tr'), {
          id, email, name, status, numLists: list.length,
        });
      });
    });
  });

  it('Sorts subscribers', () => {
    const asc = [3, 4, 5, 6, 7, 8];
    const desc = [8, 7, 6, 5, 4, 3];

    // Only e-mail and created date are sortable in the redesigned table.
    const cases = ['cy-email', 'cy-created_at'];

    cases.forEach((c) => {
      cy.sortTable(`thead th.${c}`, asc);
      cy.wait(250);
      cy.sortTable(`thead th.${c}`, desc);
      cy.wait(250);
    });
  });
});

describe('Domain blocklist', () => {
  it('Opens settings page', () => {
    cy.resetDB();
  });

  it('Add domains to blocklist', () => {
    cy.loginAndVisit('/admin/settings/privacy');
    cy.get('textarea[name="privacy.domain_blocklist"]').clear().type('ban.net\n\nBaN.OrG\n\nban.com\n\n');
    cy.get('[data-cy=btn-save]').click();
    cy.waitForBackend();
  });

  it('Try subscribing via public page', () => {
    cy.visit(`${apiUrl}/subscription/form`);
    cy.get('input[name=email]').clear().type('test@noban.net');
    cy.get('button[type=submit]').click();
    cy.get('h2').contains('Subscribe');

    cy.visit(`${apiUrl}/subscription/form`);
    cy.get('input[name=email]').clear().type('test@ban.net');
    cy.get('button[type=submit]').click();
    cy.get('h2').contains('Error');
  });

  // Post to the admin API.
  it('Try via admin API', () => {
    cy.wait(1000);

    // Add non-banned domain.
    cy.request({
      method: 'POST',
      url: `${apiUrl}/api/subscribers`,
      failOnStatusCode: true,
      body: {
        email: 'test1@noban.net', name: 'test', lists: [1], status: 'enabled',
      },
    }).should((response) => {
      expect(response.status).to.equal(200);
    });

    // Add banned domain.
    cy.request({
      method: 'POST',
      url: `${apiUrl}/api/subscribers`,
      failOnStatusCode: false,
      body: {
        email: 'test1@ban.com', name: 'test', lists: [1], status: 'enabled',
      },
    }).should((response) => {
      expect(response.status).to.equal(400);
    });

    // Modify an existing subscriber to a banned domain.
    cy.request({
      method: 'PUT',
      url: `${apiUrl}/api/subscribers/1`,
      failOnStatusCode: false,
      body: {
        email: 'test3@ban.org', name: 'test', lists: [1], status: 'enabled',
      },
    }).should((response) => {
      expect(response.status).to.equal(400);
    });
  });

  it('Try via import', () => {
    cy.loginAndVisit('/admin/subscribers/import');

    // Step 1: pick the file; the options only appear after.
    cy.fixture('subs-domain-blocklist.csv').then((data) => {
      cy.get('input[type="file"]').attachFile({
        fileContent: data.toString(),
        fileName: 'subs.csv',
        mimeType: 'text/csv',
      });
    });

    // Step 2: options.
    pickFirstList(() => cy.get('.imp-options'));
    cy.get('[data-cy=btn-import]').click();

    // Step 3: progress and log.
    cy.get('[data-cy=import-status]', { timeout: 10000 }).should('have.attr', 'data-status', 'finished');
    cy.get('.log-view').should('contain', 'ban1-import@BAN.net').and('contain', 'ban2-import@ban.ORG');

    // Clear the finished import.
    cy.get('[data-cy=btn-import-done]').click();
    cy.get('input[type="file"]').should('exist');
  });

  it('Clear blocklist and try', () => {
    cy.loginAndVisit('/admin/settings/privacy');
    cy.get('textarea[name="privacy.domain_blocklist"]').clear();
    cy.get('[data-cy=btn-save]').click();
    cy.waitForBackend();

    // Add banned domain.
    cy.request({
      method: 'POST',
      url: `${apiUrl}/api/subscribers`,
      failOnStatusCode: true,
      body: {
        email: 'test4@BAN.com', name: 'test', lists: [1], status: 'enabled',
      },
    }).should((response) => {
      expect(response.status).to.equal(200);
    });

    // Modify an existing subscriber to a banned domain.
    cy.request({
      method: 'PUT',
      url: `${apiUrl}/api/subscribers/1`,
      failOnStatusCode: true,
      body: {
        email: 'test4@BAN.org', name: 'test', lists: [1], status: 'enabled',
      },
    }).should((response) => {
      expect(response.status).to.equal(200);
    });
  });
});
