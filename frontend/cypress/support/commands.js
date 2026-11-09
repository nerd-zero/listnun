import 'cypress-file-upload';
import 'cypress-wait-until';

Cypress.Commands.add('resetDB', () => {
  // Although cypress clearly states that a webserver should not be run
  // from within it, listnun is killed, the DB reset, and run again
  // in the background. If the DB is reset without restarting listnun,
  // the live Postgres connections in the app throw errors because the
  // schema changes midway.
  cy.task('resetServer');
  cy.waitForBackend();
});

Cypress.Commands.add('resetDBBlank', () => {
  cy.task('resetServer', { blank: true });
  cy.waitForBackend();
});

// Takes a sortable column header selector (the column's header-class, e.g.
// `thead th.cy-name`), clicks it to sort the PrimeVue table, then compares
// the [data-id] of each row against the given IDs, asserting the order.
Cypress.Commands.add('sortTable', (theadSelector, ordIDs) => {
  cy.get(theadSelector).click();
  cy.wait(250);
  cy.get('.p-datatable-tbody > tr [data-id]').each(($el, index) => {
    expect(ordIDs[index]).to.equal(parseInt($el.attr('data-id'), 10));
  });
});

Cypress.Commands.add('loginAndVisit', (url) => {
  cy.visit(`/admin/login?next=${url}`);

  const username = Cypress.env('LISTNUN_ADMIN_USER') || 'admin';
  const password = Cypress.env('LISTNUN_ADMIN_PASSWORD') || 'listnun123';

  cy.get('input[name=username]').invoke('val', username);
  cy.get('input[name=password]').invoke('val', password);

  // The page also has a show-password button and, with OIDC, a second
  // submit; the first submit button signs in with the password.
  cy.get('form button[type=submit]').first().click();

  // Wait for login redirection to complete.
  if (url.includes('/admin')) {
    cy.get('[data-cy=user-menu]', { timeout: 10000 }).should('exist');
  } else {
    cy.wait(1000);
  }
});

// Clicks sidebar entries by their data-cy (the nav item's key or cy),
// e.g. cy.clickMenu('lists', 'all-lists'). A group button toggles its
// sub-items, so it is only clicked when the next entry isn't showing yet.
Cypress.Commands.add('clickMenu', (...selectors) => {
  selectors.forEach((s, i) => {
    const next = selectors[i + 1];
    cy.get(`.ln-nav [data-cy="${s}"]`).first().then(($el) => {
      const isGroup = $el.is('button[aria-expanded]');
      if (isGroup && next && $el.attr('aria-expanded') === 'true') return;
      cy.wrap($el).click();
    });
  });
});

// The currently open PrimeVue dialog (DialogForm, ConfirmAction, ...).
Cypress.Commands.add('dialog', () => cy.get('.p-dialog:visible').last());

// Clicks the confirm button of the open ConfirmAction.
Cypress.Commands.add('confirmAction', (typed) => {
  if (typed) {
    cy.get('.ln-confirm:visible input').type(typed);
  }
  cy.get('.ln-confirm:visible [data-cy=btn-confirm]').click();
});

// Rows of the PrimeVue DataTable on the page (or within the subject).
Cypress.Commands.add('rows', { prevSubject: 'optional' }, (subject) => (
  subject ? cy.wrap(subject).find('.p-datatable-tbody > tr') : cy.get('.p-datatable-tbody > tr')
));

// Picks an option in a PrimeVue Select. `sel` is a selector for the
// Select's root or its input (e.g. '#user-role' via input-id).
Cypress.Commands.add('pvSelect', (sel, label) => {
  cy.get(sel).closest('.p-select').click();
  cy.get('.p-select-overlay .p-select-option').contains(label).click();
});

// Waits for and returns the latest toast message.
Cypress.Commands.add('toast', () => cy.get('.p-toast-message', { timeout: 5000 }).last());

// https://www.nicknish.co/blog/cypress-targeting-elements-inside-iframes
Cypress.Commands.add('iframe', { prevSubject: 'element' }, ($iframe, callback = () => { }) => cy
  .wrap($iframe)
  .should((iframe) => expect(iframe.contents().find('body')).to.exist)
  .then((iframe) => cy.wrap(iframe.contents().find('body')))
  .within({}, callback));

Cypress.Commands.add('waitForBackend', () => {
  // The server restarts after a 500ms delay on settings change.
  // Wait for the server to go down
  cy.wait(1000);

  // Keep polling the public /health endpoint until the (new) server
  // is live. Use fetch() as cy.request() throws on ECONNREFUSED even with failOnStatusCode:false.
  cy.waitUntil(
    () => cy.wrap(null, { log: false }).then(() => fetch('/health')
      .then((res) => res.status === 200)
      .catch(() => false)),
    {
      timeout: 60000,
      interval: 2000,
    },
  );
});

Cypress.on('uncaught:exception', (err) => {
  if (Object.prototype.hasOwnProperty.call(err, 'request')) {
    return false;
  }

  return true;
});
