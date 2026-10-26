// Step 1 of the import flow: upload the file. Step 2 (match columns)
// appears once the server has read it.
const uploadCSV = (name = 'subs.csv', content = null) => {
  const attach = (data) => cy.get('input[type="file"]').attachFile({
    fileContent: data, fileName: name, mimeType: 'text/csv',
  });
  if (content) attach(content);
  else cy.fixture('subs.csv').then((data) => attach(data.toString()));
  cy.contains('h2', 'Match your columns').should('exist');
};

// Picks the first list in the options step's list picker.
const pickFirstList = () => {
  cy.get('.imp-lists .p-autocomplete-dropdown').click();
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
      { mode: 'Add them as subscribers', update: true, count: 102 },
      { mode: 'Add them as subscribers', update: false, count: 102 },
      { mode: 'Block these addresses', update: false, count: 102 },
    ];

    cases.forEach((c) => {
      uploadCSV();
      cy.contains('button', 'Continue').click();

      cy.contains('.p-togglebutton', c.mode).click();
      if (c.mode === 'Add them as subscribers') {
        pickFirstList();
        if (c.update) cy.contains('.imp-choice', 'Update their details').click();
      }
      cy.contains('button', 'Review import').click();
      cy.get('[data-cy=btn-import]').click();

      cy.get('[data-cy=import-status]', { timeout: 10000 }).should('have.attr', 'data-status', 'finished');

      // Clear the finished import so the next one can start.
      cy.get('[data-cy=btn-import-done]').click();
      cy.get('input[type="file"]').should('exist');

      // 100 (+2 default) subscribers.
      cy.loginAndVisit('/admin/subscribers');
      cy.get('.ln-page-header .ln-page-count').should(($el) => {
        expect(parseInt($el.text().trim(), 10)).to.equal(c.count);
      });

      cy.loginAndVisit('/admin/subscribers/import');
    });
  });

  it('Needs an email column', () => {
    cy.resetDB();
    cy.loginAndVisit('/admin/subscribers/import');

    uploadCSV('people.csv', 'who,city\nAna,Harare\n');
    cy.contains('Choose the email column').should('exist');
    cy.contains('button', 'Continue').should('be.disabled');
  });
});
