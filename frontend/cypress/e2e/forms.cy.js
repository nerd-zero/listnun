const apiUrl = Cypress.env('apiUrl');

describe('Forms', () => {
  it('Opens forms page', () => {
    cy.resetDB();
    cy.loginAndVisit('/admin/lists/forms');
    cy.contains('No signup forms yet');
  });

  it('Creates and publishes a form', () => {
    cy.contains('button', 'New form').click();
    cy.get('[data-cy=form-name]').clear().type('Newsletter signup');

    // A form can't go live without a list.
    cy.get('[data-cy=form-list]').contains('Opt-in list').click();
    cy.get('[data-cy=btn-publish]').click();

    cy.get('[data-cy=form-card]').should('contain', 'Newsletter signup').and('contain', 'Live');
    // The page address comes from the name the form was published with.
    cy.get('[data-cy=form-url]').should('have.attr', 'href').and('contain', '/f/newsletter-signup');
  });

  it('Signs up on the hosted page', () => {
    cy.visit(`${apiUrl}/f/newsletter-signup`);
    cy.get('input[name=email]').type('form-signup@test.com');
    cy.get('[data-cy=btn-subscribe]').click();
    cy.contains(/confirm|subscribed/i);

    cy.request(`${apiUrl}/api/subscribers?per_page=all`).should((response) => {
      const sub = response.body.data.results.find((s) => s.email === 'form-signup@test.com');
      expect(sub, 'subscriber from the form').to.not.equal(undefined);
      expect(sub.lists.map((l) => l.name)).to.include('Opt-in list');
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
