const apiUrl = Cypress.env('apiUrl');

// Steps of the campaign editor, in order.
const STEP_SCHEDULE = 2;
const STEP_REVIEW = 3;

// Opens the n-th campaign on the campaigns page.
function openCampaign(n) {
  cy.rows().eq(n).find('a.ln-row-title').click();
  cy.location('pathname').should('match', /\/campaigns\/\d+$/);
  cy.get('[data-campaign-id]').should('exist');
}

// Clones the first campaign on the campaigns page with the given name.
function cloneFirst(name) {
  cy.loginAndVisit('/admin/campaigns/all');
  cy.get('[data-cy=btn-clone]').first().click();
  cy.dialog().within(() => {
    cy.get('#clone-name').clear().type(name);
    cy.get('[data-cy=btn-save]').click();
  });

  // Cloning opens the new campaign.
  cy.location('pathname').should('match', /\/campaigns\/\d+$/);
  cy.get('.camp-bar h1').should('contain', name);
}

describe('Archive', () => {
  it('Opens campaigns page', () => {
    cy.resetDB();
    cy.loginAndVisit('/admin/campaigns/all');
    cy.wait(500);
  });

  it('Clones campaign', () => {
    cloneFirst('clone');
    cloneFirst('clone2');
    cy.visit('/admin/campaigns/all');
  });

  it('Starts campaigns', () => {
    [0, 1].forEach((n) => {
      openCampaign(n);

      // Start from the review step.
      cy.get('.camp-steps .camp-step').eq(STEP_REVIEW).click();
      cy.get('.camp-bar [data-cy=btn-start]').click();
      cy.confirmAction();

      // Starting returns to the campaigns page.
      cy.location('pathname').should('match', /\/campaigns$/);
    });
    cy.wait(1000);
  });

  it('Enables archive on one campaign (no slug)', () => {
    cy.loginAndVisit('/admin/campaigns/all');
    openCampaign(0);

    // The archive settings are on the schedule step.
    cy.get('.camp-steps .camp-step').eq(STEP_SCHEDULE).click();
    cy.get('[data-cy=btn-archive] input').check({ force: true });
    cy.get('[data-cy=archive-slug]').clear();
    cy.get('[data-cy=archive-meta]').clear()
      .type('{"email": "archive@domain.com", "name": "Archive", "attribs": { "city": "Bengaluru"}}', { parseSpecialCharSequences: false });
    cy.get('[data-cy=btn-save]').click();
    cy.toast();
  });

  it('Enables archive on one campaign', () => {
    cy.loginAndVisit('/admin/campaigns/all');
    openCampaign(1);

    cy.get('.camp-steps .camp-step').eq(STEP_SCHEDULE).click();
    cy.get('[data-cy=btn-archive] input').check({ force: true });
    cy.get('[data-cy=archive-slug]').clear().type('my-archived-campaign');
    cy.get('[data-cy=btn-save]').click();
    cy.toast();
  });

  it('Opens campaign archive page', () => {
    for (let i = 0; i < 2; i += 1) {
      cy.loginAndVisit(`${apiUrl}/archive`);
      cy.get('[data-cy=archive-item]').eq(i).click();
      cy.wait(250);
      if (i === 0) {
        cy.get('h3').contains('Hi Archive!');
        cy.get('p').eq(0).contains('Bengaluru');
      } else {
        cy.get('h3').contains('Hi Subscriber!');
      }
    }
  });
});
