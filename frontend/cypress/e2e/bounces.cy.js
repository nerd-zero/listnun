const apiUrl = Cypress.env('apiUrl');

describe('Bounces', () => {
  it('Enable bounces', () => {
    cy.resetDB();

    cy.loginAndVisit('/admin/settings/bounces');
    cy.get('[data-cy=btn-enable-bounce] input[role=switch]').check({ force: true });
    cy.get('[data-cy=btn-enable-bounce-webhook] input[role=switch]').check({ force: true });

    // The last action Select (complaint bounces): pick the 'delete' option.
    cy.get('[name="bounce.action"]').last().closest('.p-select').click();
    cy.get('.p-select-overlay .p-select-option').contains('Delete').click();

    cy.get('[data-cy=btn-save]').click();

    cy.waitForBackend();
  });

  it('Post bounces', () => {
    cy.loginAndVisit('/admin/subscribers/bounces');

    // Get subscribers.
    let subs = [];
    cy.request(`${apiUrl}/api/subscribers`).then((resp) => {
      subs = resp.body.data.results;
    }).then(() => {
      // Register soft bounces do nothing.
      let sub = {};
      cy.request('POST', `${apiUrl}/webhooks/bounce`, { source: 'api', type: 'soft', email: subs[0].email });
      cy.request('POST', `${apiUrl}/webhooks/bounce`, { source: 'api', type: 'soft', email: subs[0].email });
      cy.request(`${apiUrl}/api/subscribers/${subs[0].id}`).then((resp) => {
        sub = resp.body.data;
      }).then(() => {
        cy.expect(sub.status).to.equal('enabled');
      });

      // Hard bounces blocklist.
      cy.request('POST', `${apiUrl}/webhooks/bounce`, { source: 'api', type: 'hard', email: subs[0].email });
      cy.request('POST', `${apiUrl}/webhooks/bounce`, { source: 'api', type: 'hard', email: subs[0].email });

      cy.request(`${apiUrl}/api/subscribers/${subs[0].id}`).then((resp) => {
        sub = resp.body.data;
      }).then(() => {
        cy.expect(sub.status).to.equal('blocklisted');
      });

      // Complaint bounces delete.
      cy.request('POST', `${apiUrl}/webhooks/bounce`, { source: 'api', type: 'complaint', email: subs[1].email });
      cy.request('POST', `${apiUrl}/webhooks/bounce`, { source: 'api', type: 'complaint', email: subs[1].email });
      cy.request({ url: `${apiUrl}/api/subscribers/${subs[1].id}`, failOnStatusCode: false }).then((resp) => {
        expect(resp.status).to.eq(400);
      });

      // The remaining bounces are listed against the blocklisted subscriber.
      cy.loginAndVisit('/admin/subscribers/bounces');
      cy.rows().should('contain', subs[0].email);
      cy.get('.p-datatable-tbody [data-cy=blocklisted]').should('exist');
    });
  });
});
