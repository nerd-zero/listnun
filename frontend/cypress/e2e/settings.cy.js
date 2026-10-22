const apiUrl = Cypress.env('apiUrl');
const rootURL = 'http://127.0.0.1:9000';
const faveURL = 'http://127.0.0.1:9000/public/static/logo.png';

describe('Settings', () => {
  it('Opens settings page', () => {
    cy.resetDB();
    cy.loginAndVisit('/admin/settings/general');
  });

  it('Changes some settings', () => {
    // General.
    cy.get('[data-cy=subnav-general]').click();
    cy.get('input[name="app.root_url"]').clear().type(rootURL);
    cy.get('input[name="app.favicon_url"]').type(faveURL);

    // Performance. All sections share one form, so edits survive switching.
    cy.get('[data-cy=subnav-performance]').click();
    cy.get('input[name="app.concurrency"]').clear().type('9').blur();

    // Enable / disable SMTP and delete one.
    cy.get('[data-cy=subnav-smtp]').click();
    cy.get('[data-cy=btn-enable-smtp]').eq(1).find('input[role=switch]').check({ force: true });
    cy.get('[data-cy=btn-delete-smtp]').first().click();
    // $utils.confirm() opens the global PrimeVue ConfirmDialog.
    cy.get('.p-confirmdialog-accept-button').click();
    cy.get('[data-cy=btn-enable-smtp]').should('have.length', 1);

    // Save from the unsaved-changes bar.
    cy.get('.settings-unsaved [data-cy=btn-save]').click();
    cy.wait(500);

    cy.waitForBackend();
  });

  it('Verify settings change', () => {
    // Verify the changes.
    cy.request(`${apiUrl}/api/settings`).should((response) => {
      const { data } = response.body;
      expect(data['app.root_url']).to.equal(rootURL);
      expect(data['app.favicon_url']).to.equal(faveURL);
      expect(data['app.concurrency']).to.equal(9);

      expect(data.smtp.length).to.equal(1);
      expect(data.smtp[0].enabled).to.equal(true);
    });
  });
});
