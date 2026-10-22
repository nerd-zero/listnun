// Step 1 of the import flow: pick the CSV. The file input is hidden; the
// options (step 2) only render once a file is picked.
const attachCSV = () => {
  cy.fixture('subs.csv').then((data) => {
    cy.get('input[type="file"]').attachFile({
      fileContent: data.toString(),
      fileName: 'subs.csv',
      mimeType: 'text/csv',
    });
  });
  cy.get('.imp-options').should('exist');
};

// Picks the first list in the options step's ListSelector.
const pickFirstList = () => {
  cy.get('.imp-options .list-selector .p-autocomplete-dropdown').click();
  cy.get('.p-autocomplete-overlay .p-autocomplete-option').first().click();
  cy.get('.p-autocomplete-overlay').should('not.exist');
};

describe('Import', () => {
  it('Opens import page', () => {
    cy.resetDB();
    cy.loginAndVisit('/admin/subscribers/import');
  });

  it('Imports subscribers', () => {
    const cases = [
      {
        chkMode: 'subscribe', status: 'enabled', chkSubStatus: 'unconfirmed', subStatus: 'unconfirmed', overwrite: true, count: 102,
      },
      {
        chkMode: 'subscribe', status: 'enabled', chkSubStatus: 'confirmed', subStatus: 'confirmed', overwrite: true, count: 102,
      },
      {
        chkMode: 'subscribe', status: 'enabled', chkSubStatus: 'unconfirmed', subStatus: 'confirmed', overwrite: false, count: 102,
      },
      {
        chkMode: 'blocklist', status: 'blocklisted', chkSubStatus: 'unsubscribed', subStatus: 'unsubscribed', overwrite: false, count: 102,
      },
    ];

    cases.forEach((c) => {
      attachCSV();

      cy.get(`[data-cy=check-${c.chkMode}] input`).check({ force: true });
      cy.get(`[data-cy=check-${c.chkSubStatus}] input`).check({ force: true });

      if (c.overwrite) {
        cy.get('[data-cy=overwrite-user-info] input').check({ force: true });
        cy.get('[data-cy=overwrite-sub-status] input').check({ force: true });
      }

      if (c.status === 'enabled') {
        pickFirstList();
      }

      cy.get('[data-cy=btn-import]').click();

      // Overwriting subscription statuses asks for confirmation first.
      if (c.overwrite) {
        cy.confirmAction();
      }

      cy.get('[data-cy=import-status]', { timeout: 10000 }).should('have.attr', 'data-status', 'finished');

      // Clear the finished import so the next one can start.
      cy.get('[data-cy=btn-import-done]').click();
      cy.get('input[type="file"]').should('exist');

      // Verify that 100 (+2 default) subs are imported.
      cy.loginAndVisit('/admin/subscribers');
      cy.get('.ln-page-header .ln-page-count').should(($el) => {
        expect(parseInt($el.text().trim(), 10)).to.equal(c.count);
      });

      cy.loginAndVisit('/admin/subscribers/import');
    });
  });

  it('Imports subscribers incorrectly', () => {
    cy.wait(1000);
    cy.resetDB();
    cy.wait(1000);
    cy.loginAndVisit('/admin/subscribers/import');

    attachCSV();
    pickFirstList();
    cy.get('input[name=delim]').clear().type('|');

    cy.get('[data-cy=btn-import]').click();
    // A wrong delimiter ends the import as failed (or stopped).
    cy.get('[data-cy=import-status]', { timeout: 10000 })
      .should('have.attr', 'data-status')
      .and('match', /^(failed|stopped)$/);
  });
});
