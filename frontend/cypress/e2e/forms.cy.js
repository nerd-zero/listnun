const apiUrl = Cypress.env('apiUrl');

describe('Forms', () => {
  it('Opens forms page', () => {
    cy.resetDB();
    cy.loginAndVisit('/admin/lists/forms');
  });

  it('Checks public lists', () => {
    cy.get('[data-cy=lists] [data-cy=list]')
      .should('contain', 'Opt-in list')
      .its('length')
      .should('eq', 1);

    // Every public list is preselected, so the form HTML is shown straight away.
    cy.get('[data-cy=lists] [data-cy=list] input[type=checkbox]').should('be.checked');
    cy.get('[data-cy=form] [data-cy=html]').should('exist');
    cy.get('[data-cy=form] [data-cy=no-selection]').should('not.exist');
  });

  it('Selects public list', () => {
    cy.request(`${apiUrl}/api/lists?per_page=all`).then((response) => {
      const { uuid } = response.body.data.results.find((l) => l.name === 'Opt-in list');

      // Deselect the list: the HTML goes away.
      cy.get('[data-cy=lists] [data-cy=list] input[type=checkbox]').uncheck({ force: true });
      cy.get('[data-cy=form] [data-cy=html]').should('not.exist');
      cy.get('[data-cy=form] [data-cy=no-selection]').should('exist');
      cy.get('[data-cy=form] [data-cy=preview]').should('have.attr', 'srcdoc').and('not.contain', uuid);

      // Select it again: the list's UUID appears in the form HTML.
      cy.get('[data-cy=lists] [data-cy=list] input[type=checkbox]').check({ force: true });
      cy.get('[data-cy=form] [data-cy=html]').should('exist');
      cy.get('[data-cy=form] [data-cy=preview]').should('have.attr', 'srcdoc').and('contain', uuid);
    });
  });

  it('Subscribes from public form page', () => {
    // Create a public test list.
    cy.request('POST', `${apiUrl}/api/lists`, { name: 'test-list', type: 'public', optin: 'single' });

    // Open the public page and subscribe to alternating lists multiple times.
    // Every list starts checked; unchecking one subscribes to the other.
    // There should be no errors and two new subscribers should be subscribed to two lists.
    for (let i = 0; i < 2; i += 1) {
      for (let j = 0; j < 2; j += 1) {
        cy.loginAndVisit(`${apiUrl}/subscription/form`);
        cy.get('input[name=email]').clear().type(`test${i}@test.com`);
        cy.get('input[name=name]').clear().type(`test${i}`);
        cy.get('ul[data-cy=lists] input[type=checkbox]').eq(j).click();
        cy.get('[data-cy=btn-subscribe]').click();
        cy.wait(250);
        cy.get('main.wrap').contains(/has been sent|successfully|retry/); // If SMTP is not configured, it shows retry message.
      }
    }

    // Verify form subscriptions.
    cy.request(`${apiUrl}/api/subscribers`).should((response) => {
      const { data } = response.body;

      // Two new + two dummy subscribers that are there by default.
      expect(data.total).to.equal(4);

      // The two new subscribers should each have two list subscriptions.
      for (let i = 0; i < 2; i += 1) {
        expect(data.results.find((s) => s.email === `test${i}@test.com`).lists.length).to.equal(2);
      }
    });
  });

  it('Unsubscribes', () => {
    // Add all lists to the dummy campaign.
    cy.request('PUT', `${apiUrl}/api/campaigns/1`, { lists: [2] });

    cy.request('GET', `${apiUrl}/api/subscribers`).then((response) => {
      const subUUID = response.body.data.results[0].uuid;

      cy.request('GET', `${apiUrl}/api/campaigns`).then((campResp) => {
        const campUUID = campResp.body.data.results[0].uuid;
        cy.loginAndVisit(`${apiUrl}/subscription/${campUUID}/${subUUID}`);
      });
    });

    cy.wait(500);

    // Unsubscribe from one list. The page also has export/wipe buttons, so
    // target the unsubscribe button.
    cy.get('#btn-unsub').click();
    cy.request('GET', `${apiUrl}/api/subscribers`).then((response) => {
      const { data } = response.body;
      expect(data.results[0].lists.find((s) => s.id === 2).subscription_status).to.equal('unsubscribed');
      expect(data.results[0].lists.find((s) => s.id === 3).subscription_status).to.equal('unconfirmed');
    });

    // Go back.
    cy.url().then((u) => {
      cy.loginAndVisit(u);
    });

    // Unsubscribe from all.
    cy.get('#privacy-blocklist').click();
    cy.get('#btn-unsub').click();

    cy.request('GET', `${apiUrl}/api/subscribers`).then((response) => {
      const { data } = response.body;
      expect(data.results[0].status).to.equal('blocklisted');
      expect(data.results[0].lists.find((s) => s.id === 2).subscription_status).to.equal('unsubscribed');
      expect(data.results[0].lists.find((s) => s.id === 3).subscription_status).to.equal('unsubscribed');
    });
  });

  it('Manages subscription preferences', () => {
    cy.request('GET', `${apiUrl}/api/subscribers`).then((response) => {
      const subUUID = response.body.data.results[1].uuid;

      cy.request('GET', `${apiUrl}/api/campaigns`).then((campResp) => {
        const campUUID = campResp.body.data.results[0].uuid;
        cy.loginAndVisit(`${apiUrl}/subscription/${campUUID}/${subUUID}`);
        cy.get('[data-cy=link-manage]').click();
      });
    });

    // Change name and unsubscribe from one list.
    cy.get('#sub-name').clear().type('new-name');
    cy.get('ul[data-cy=lists] input[type=checkbox]').first().click();
    cy.get('#btn-unsub').click();

    cy.request('GET', `${apiUrl}/api/subscribers`).then((response) => {
      const { data } = response.body;
      expect(data.results[1].name).to.equal('new-name');
      expect(data.results[1].lists.find((s) => s.id === 2).subscription_status).to.equal('unsubscribed');
      expect(data.results[1].lists.find((s) => s.id === 3).subscription_status).to.equal('unconfirmed');
    });
  });
});
