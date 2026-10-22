const apiUrl = Cypress.env('apiUrl');

// Goes to the lists page through the sidebar. The Lists group button is a
// toggle, so it is only clicked when its sub-items are not already open.
function goToLists() {
  cy.get('.ln-nav').first().then(($nav) => {
    if (!$nav.find('[data-cy=all-lists]').length) {
      cy.get('.ln-nav [data-cy=lists]').first().click();
    }
  });
  cy.get('.ln-nav [data-cy=all-lists]').first().click();
}

// Opens a row's popup menu and clicks the item with the given label.
function rowMenu($row, label) {
  cy.wrap($row).find('[data-cy=btn-row-menu]').click();
  cy.get('.p-menu:visible .p-menu-item').contains(label).click();
}

// Fills and submits the open ListForm dialog.
function fillListForm({
  name, type, optin, tags, description,
}) {
  cy.dialog().within(() => {
    cy.get('#list-name').clear().type(name);
    cy.get(`#list-type-${type}`).check({ force: true });
    if (optin === 'double') {
      cy.get('#list-optin').check({ force: true });
    } else {
      cy.get('#list-optin').uncheck({ force: true });
    }
    cy.get('#list-tags input').type(tags);
    cy.get('#list-desc').clear().type(description);
    cy.get('[data-cy=btn-save]').click();
  });
  cy.get('.p-dialog').should('not.exist');
}

describe('Lists', () => {
  it('Opens lists page', () => {
    cy.resetDB();
    cy.loginAndVisit('/admin/lists');
  });

  it('Counts subscribers in default lists', () => {
    cy.get('.p-datatable-tbody [data-cy=subscribers]').first().contains('1');
  });

  it('Creates campaign for list', () => {
    cy.get('.p-datatable-tbody a[data-cy=btn-campaign]').first().click();
    cy.location('pathname').should('contain', '/campaigns/new');
    cy.get('.list-tags .p-tag').contains('Default list');

    // Nothing was typed into the new campaign, so leaving asks nothing.
    goToLists();
    cy.location('pathname').should('match', /\/lists$/);
  });

  it('Creates opt-in campaign for list', () => {
    cy.contains('.p-datatable-tbody > tr', 'Opt-in list').then(($tr) => rowMenu($tr, 'Send opt-in campaign'));
    cy.confirmAction();
    cy.location('pathname').should('contain', '/campaigns/2');

    goToLists();
    // The editor may normalise the drafted body and ask to save before leaving.
    cy.get('body').then(($body) => {
      if ($body.find('.ln-confirm:visible').length) cy.confirmAction();
    });
    cy.location('pathname').should('match', /\/lists$/);
  });

  it('Checks individual subscribers in lists', () => {
    const subs = [{ listID: 1, email: 'john@example.com' },
      { listID: 2, email: 'anon@example.com' }];

    // Click on each list on the lists page, go the subscribers page
    // for that list, and check the subscriber details.
    subs.forEach((s, n) => {
      cy.get('.p-datatable-tbody [data-cy=subscribers]').eq(n).click();
      cy.location('pathname').should('contain', `/subscribers/lists/${s.listID}`);
      cy.rows().should('have.length', 1).first().should('contain', s.email);
      goToLists();
    });
  });

  it('Edits lists', () => {
    // Open the edit popup and edit the default lists.
    [0, 1].forEach((n) => {
      cy.get('.p-datatable-tbody [data-cy=btn-edit]').eq(n).click();
      fillListForm({
        name: `list-${n}`, type: 'public', optin: 'double', tags: `tag${n}{enter}`, description: `desc${n}`,
      });
    });

    // Confirm the edits.
    cy.rows().each(($el, n) => {
      cy.wrap($el).find('[data-cy=list-name]').contains(`list-${n}`);
      cy.wrap($el).find('.lists-tags')
        .should('contain', 'test')
        .and('contain', `tag${n}`);
    });
  });

  it('Deletes lists', () => {
    // Delete all visible lists.
    [0, 1].forEach(() => {
      cy.rows().first().then(($tr) => rowMenu($tr, 'Delete'));
      cy.confirmAction();
      cy.get('.ln-confirm').should('not.exist');
    });

    // Confirm deletion.
    cy.get('.p-datatable-empty-message');
  });

  // Add new lists.
  it('Adds new lists', () => {
    // Open the list form and create lists of multiple type/optin combinations.
    const types = ['private', 'public'];
    const optin = ['single', 'double'];

    let n = 0;
    types.forEach((t) => {
      optin.forEach((o) => {
        const name = `list-${t}-${o}-${n}`;

        cy.get('[data-cy=btn-new]').click();
        fillListForm({
          name, type: t, optin: o, tags: `tag${n}{enter}${t}{enter}${o}{enter}`, description: `desc-${t}-${n}`,
        });

        // Confirm the addition by inspecting the newly created list row.
        cy.rows().eq(n).within(() => {
          cy.get('[data-cy=list-name]').contains(name);
          cy.get(`[data-cy=type-${t}]`);
          cy.get(`[data-cy=optin-${o}]`);
        });
        n += 1;
      });
    });
  });

  it('Searches lists', () => {
    cy.get('[data-cy=query]').clear().type('list-public-single-2{enter}');
    cy.rows().should('have.length', 1);
    cy.get('.p-datatable-tbody [data-cy=list-name]').first().contains('list-public-single-2');
    cy.get('[data-cy=query]').clear().type('{enter}');
    cy.rows().should('have.length', 4);
  });

  // Sort lists by clicking on various headers. At this point, there should be four
  // lists with IDs = [3, 4, 5, 6]. Sort the items be columns and match them with
  // the expected order of IDs.
  it('Sorts lists', () => {
    cy.sortTable('thead th.cy-name', [4, 3, 6, 5]);
    cy.sortTable('thead th.cy-name', [5, 6, 3, 4]);

    cy.sortTable('thead th.cy-type', [3, 4, 5, 6]);
    cy.sortTable('thead th.cy-type', [6, 5, 4, 3]);

    cy.sortTable('thead th.cy-updated_at', [3, 4, 5, 6]);
    cy.sortTable('thead th.cy-updated_at', [6, 5, 4, 3]);
  });

  it('Opens forms page', () => {
    cy.loginAndVisit(`${apiUrl}/subscription/form`);
    cy.get('ul[data-cy=lists] li').its('length').should('eq', 2);

    const cases = [
      { name: 'list-public-single-2', description: 'desc-public-2' },
      { name: 'list-public-double-3', description: 'desc-public-3' },
    ];

    cases.forEach((c, n) => {
      cy.get('ul[data-cy=lists] li').eq(n).within(() => {
        cy.get('label').contains(c.name);
        cy.get('.description').contains(c.description);
      });
    });
  });

  it('Bulk deletes lists', () => {
    // Create 30 in a loop.
    for (let i = 0; i < 30; i += 1) {
      cy.request('POST', `${apiUrl}/api/lists`, { name: `test-list-${i}`, type: 'public', optin: 'single' });
    }

    cy.loginAndVisit('/admin/lists');

    // Bulk delete with the `all` flag. Deleting every matching list asks
    // for the count to be typed in.
    cy.window().scrollTo('top');
    cy.rows().should('have.length.greaterThan', 1);
    cy.get('.p-datatable-thead .p-checkbox input').click({ force: true });
    cy.get('a[data-cy=select-all-lists]').click();
    cy.get('[data-cy=btn-delete-lists]').click();
    cy.get('.ln-confirm:visible code').invoke('text').then((n) => cy.confirmAction(n.trim()));
    cy.get('.p-datatable-empty-message');

    // Bulk delete with the selected IDs.
    // Create 5 lists in a loop.
    for (let i = 0; i < 5; i += 1) {
      cy.request('POST', `${apiUrl}/api/lists`, { name: `test-list-bulk-${i}`, type: 'public', optin: 'single' });
    }

    cy.visit('/admin/lists');
    cy.rows().should('have.length', 5);
    cy.get('.p-datatable-thead .p-checkbox input').click({ force: true });
    cy.get('[data-cy=btn-delete-lists]').click();
    cy.confirmAction();
    cy.get('.p-datatable-empty-message');
  });
});
