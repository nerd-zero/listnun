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

// Shows the named list in the panel.
function openList(name) {
  cy.contains('[data-cy=list-card]', name).click();
  cy.get('[data-cy=list-panel] h2').should('contain', name);
}

// Shows a list in the panel and clicks its "More actions" item with the
// given label.
function listMenu(name, label) {
  openList(name);
  cy.get('[data-cy=list-panel] [data-cy=btn-row-menu]').click();
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
    // Cards are in name order: Default list, then Opt-in list.
    cy.get('[data-cy=list-card]').first().should('contain', 'Default list').and('contain', '1');
    cy.contains('[data-cy=list-card]', 'Opt-in list').should('contain', '1 not confirmed yet');
  });

  it('Shows people waiting to confirm', () => {
    cy.get('[data-cy=optin-banner]').should('contain', '1 person signed up');
  });

  it('Creates campaign for list', () => {
    openList('Default list');
    cy.get('[data-cy=list-panel] [data-cy=btn-campaign]').click();
    cy.location('pathname').should('contain', '/campaigns/new');
    cy.get('.list-tags .p-tag').contains('Default list');

    // Nothing was typed into the new campaign, so leaving asks nothing.
    goToLists();
    cy.location('pathname').should('match', /\/lists$/);
  });

  it('Creates opt-in campaign for list', () => {
    listMenu('Opt-in list', 'Send opt-in campaign');
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
    const subs = [{ name: 'Default list', listID: 1, email: 'john@example.com' },
      { name: 'Opt-in list', listID: 2, email: 'anon@example.com' }];

    // Open each list's subscribers from its panel and check them.
    subs.forEach((s) => {
      listMenu(s.name, 'View subscribers');
      cy.location('pathname').should('contain', `/subscribers/lists/${s.listID}`);
      cy.rows().should('have.length', 1).first().should('contain', s.email);
      goToLists();
    });
  });

  it('Edits lists', () => {
    [['Default list', 0], ['Opt-in list', 1]].forEach(([name, n]) => {
      openList(name);
      cy.get('[data-cy=list-panel] [data-cy=btn-edit]').click();
      fillListForm({
        name: `list-${n}`, type: 'public', optin: 'double', tags: `tag${n}{enter}`, description: `desc${n}`,
      });
      cy.get('[data-cy=list-panel] h2').should('contain', `list-${n}`);
      cy.get('[data-cy=list-panel]').should('contain', `desc${n}`);
    });
  });

  it('Deletes lists', () => {
    ['list-0', 'list-1'].forEach((name) => {
      listMenu(name, 'Delete');
      cy.confirmAction();
      cy.get('.ln-confirm').should('not.exist');
    });

    cy.get('[data-cy=list-card]').should('not.exist');
  });

  // Add new lists.
  it('Adds new lists', () => {
    // Create lists of every type/optin combination. Only public lists
    // have a signup form tab.

    let n = 0;
    ['private', 'public'].forEach((t) => {
      ['single', 'double'].forEach((o) => {
        const name = `list-${t}-${o}-${n}`;

        cy.get('[data-cy=btn-new]').click();
        fillListForm({
          name, type: t, optin: o, tags: `tag${n}{enter}${t}{enter}${o}{enter}`, description: `desc-${t}-${n}`,
        });

        openList(name);
        cy.get('[data-cy=tab-form]').should(t === 'public' ? 'exist' : 'not.exist');
        n += 1;
      });
    });
  });

  it('Searches lists', () => {
    cy.get('[data-cy=query]').clear().type('list-public-single-2');
    cy.get('[data-cy=list-card]').should('have.length', 1).first().should('contain', 'list-public-single-2');
    cy.get('[data-cy=query]').clear();
    cy.get('[data-cy=list-card]').should('have.length', 4);
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

  it('Opens one list\'s form from its link', () => {
    cy.loginAndVisit('/admin/lists');
    openList('list-public-double-3');
    cy.get('[data-cy=list-panel] .list-panel-link .ln-mono').invoke('text').then((url) => {
      cy.visit(url.trim());
      cy.get('ul[data-cy=lists] li').should('have.length', 1).first().should('contain', 'list-public-double-3');
    });
  });
});
