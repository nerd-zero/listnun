const apiUrl = Cypress.env('apiUrl');

// The card of the template with the given name.
const card = (name) => cy.contains('.tpls-card', name);

// Clicks an item in a template card's popup menu.
const cardMenu = (name, label) => {
  card(name).find('[data-cy=btn-row-menu]').click();
  cy.get('.p-menu-overlay:visible').contains('.p-menu-item', label).click();
};

// Replaces the content of the CodeMirror 6 editor inside `sel`. CodeMirror
// keeps its view on the content node (`cmView`), which is what
// EditorView.findFromDOM() reads in @codemirror/view 6.x.
const setCode = (sel, text) => {
  cy.get(`${sel} .cm-content`).then(($el) => {
    const { view } = $el[0].cmView.rootView;
    view.dispatch({ changes: { from: 0, to: view.state.doc.length, insert: text } });
  });
};

describe('Templates', () => {
  it('Opens templates page', () => {
    cy.resetDB();
    cy.loginAndVisit('/admin/campaigns/templates');
  });

  it('Counts default templates', () => {
    // Campaign templates only; the transactional one is under Developers.
    cy.get('.tpls-card[data-cy^="template-"]').should('have.length', 3);
    card('Default campaign template').should('contain', 'Used by');
  });

  it('Clones campaign template', () => {
    cardMenu('Default campaign template', 'Clone');
    cy.get('#tpl-clone-name').clear().type('cloned campaign');
    cy.dialog().find('[data-cy=btn-save]').click();

    // Verify the newly created card.
    card('cloned campaign').should('exist');
  });

  it('Edits template', () => {
    // The editor is its own page.
    cy.get('[data-cy=template-1] [data-cy=btn-edit]').click();
    cy.location('pathname').should('eq', '/admin/campaigns/templates/1');
    cy.get('[data-cy=tpl-name]').clear().type('edited');

    const htmlBody = '<span>test</span><div class="wrap">{{ template "content" . }}</div>';
    setCode('.tpe-editor', htmlBody);
    cy.get('.tpe-frame').should('have.attr', 'srcdoc').and('contain', 'test');

    cy.get('[data-cy=btn-save]').click();
    cy.get('[data-cy=btn-save]').should('be.disabled');
    cy.get('.tpe-back').click();
    cy.get('[data-cy=template-1] .tpls-title').contains('edited');
  });

  it('Previews campaign templates', () => {
    // Edited one should have a bare body.
    cy.request(`${apiUrl}/api/templates/1/preview`).then((resp) => {
      expect(resp.body).to.contain('test');
      expect(resp.body).to.contain('Hi there');
    });

    // Cloned one should have the full template with wrap and unsubscribe.
    cy.request(`${apiUrl}/api/templates/5/preview`).then((resp) => {
      expect(resp.body).to.contain('Hi there');
      expect(resp.body).to.contain('Unsubscribe');
    });
  });

  it('Previews tx templates', () => {
    // The sample transactional template (edited under Developers).
    cy.request(`${apiUrl}/api/templates/3/preview`).then((resp) => {
      expect(resp.body).to.contain('Order number');
    });
  });

  it('Sets default', () => {
    cardMenu('cloned campaign', 'Set as default');
    cy.confirmAction();

    // The new default carries the Default tag; the old one no longer does.
    card('cloned campaign').find('.tpls-title .ln-status').should('exist');
    card('edited').find('.tpls-title .ln-status').should('not.exist');

    // The default can't be deleted; the previous default now can.
    // (Clicking the menu button again closes the popup.)
    card('cloned campaign').find('[data-cy=btn-row-menu]').click();
    cy.get('.p-menu-overlay:visible').should('not.contain', 'Delete');
    card('cloned campaign').find('[data-cy=btn-row-menu]').click();
    cy.get('.p-menu-overlay').should('not.exist');
    card('edited').find('[data-cy=btn-row-menu]').click();
    cy.get('.p-menu-overlay:visible').should('contain', 'Delete');
    card('edited').find('[data-cy=btn-row-menu]').click();
    cy.get('.p-menu-overlay').should('not.exist');
  });

  it('Deletes template', () => {
    cardMenu('Default archive template', 'Delete');
    cy.confirmAction();
    card('Default archive template').should('not.exist');

    cy.get('.tpls-card[data-cy^="template-"]').should('have.length', 3);
  });
});
