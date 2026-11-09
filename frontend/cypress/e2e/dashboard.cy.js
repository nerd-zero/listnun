describe('Dashboard', () => {
  it('Opens dashboard', () => {
    cy.resetDB();
    cy.loginAndVisit('/admin');

    // Subscriber count.
    cy.get('[data-cy=subscribers] .ln-tile-value').should('have.text', '2');

    // Message count; nothing has been sent on a fresh install.
    cy.get('[data-cy=messages] .ln-tile-value').should('have.text', '0');

    // Recent campaigns: the sample campaign, as a draft.
    cy.get('[data-cy=recent-campaigns] .dash-rows li').should('have.length', 1);
    cy.get('[data-cy=recent-campaigns] .dash-rows li').first()
      .should('contain', 'Test campaign')
      .find('.ln-status[data-status=draft]')
      .should('exist');
  });
});
