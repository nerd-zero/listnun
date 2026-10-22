const apiUrl = Cypress.env('apiUrl');

// Opens the row menu of a Users table row and clicks one of its items.
const rowMenu = (row, label) => {
  cy.wrap(row).find('[data-cy=btn-row-menu]').click();
  cy.get('.p-menu:visible .p-menu-item').contains(label).click();
};

// Picks a role in the master-detail role list and waits for its editor.
const selectRole = (name) => {
  cy.contains('[data-cy=role-item]', name).click();
  cy.get('[data-cy=role-form] #role-name').should('have.value', name);
};

// Adds a list to the list role being edited (view + manage by default).
const addList = (name) => {
  cy.pvSelect('[data-cy=role-form] [data-cy=list-select]', name);
  cy.get('[data-cy=role-form] [data-cy=btn-add-list]').click();
};

const saveRole = () => cy.get('[data-cy=role-form] [data-cy=btn-save]').click();

// Signs in through the login page (it also has a show-password button).
const login = (username, password) => {
  cy.visit('/admin/login?next=/admin/lists');
  cy.get('input[name=username]').invoke('val', username);
  cy.get('input[name=password]').invoke('val', password);
  cy.get('form button[type=submit]').first().click();
};

describe('First time user setup', () => {
  it('Sets up the superadmin user', () => {
    cy.resetDBBlank();
    cy.visit('/admin/login');

    cy.get('input[name=email]').type('super@domain');
    cy.get('input[name=username]').type('super');
    cy.get('input[name=password]').type('super123');
    cy.get('input[name=password2]').type('super123');
    cy.get('form button[type=submit]').first().click();
    cy.wait(500);
    cy.visit('/admin/users');

    cy.rows().first().then((row) => rowMenu(row, 'Edit'));
    cy.dialog().find('#user-role').closest('.p-select').should('contain', 'Super Admin');
  });
});

describe('User roles', () => {
  it('Opens user roles page', () => {
    cy.resetDB();
    cy.loginAndVisit('/admin/users/roles/users');
  });

  it('Adds new roles', () => {
    // first - the new-role defaults (no global list perms).
    cy.get('[data-cy=btn-new]').click();
    cy.get('[data-cy=role-form] #role-name').type('first');
    saveRole();
    cy.contains('[data-cy=role-item]', 'first').should('exist');

    // second - all perms.
    cy.get('[data-cy=btn-new]').click();
    cy.get('[data-cy=role-form] #role-name').type('second');
    cy.get('[data-cy=role-form] [data-cy=btn-toggle-all]').click();
    cy.get('[data-cy=role-form] input[type=checkbox]').should('be.checked');
    saveRole();
    cy.contains('[data-cy=role-item]', 'second').should('exist');
  });

  it('Edits role', () => {
    selectRole('first');
    cy.get('[data-cy=role-form] [id="perm-users:get"]').check({ force: true });
    saveRole();
    cy.toast().should('exist');
  });

  it('Deletes role', () => {
    selectRole('second');
    cy.get('[data-cy=role-form] [data-cy=btn-clone]').click();
    cy.dialog().find('[data-cy=btn-save]').click();
    cy.get('[data-cy=role-item]').should('have.length', 4);

    // The clone is selected once it's created.
    cy.get('[data-cy=role-form] #role-name').should('not.have.value', 'second');
    cy.get('[data-cy=role-form] [data-cy=btn-delete]').click();
    cy.confirmAction();

    cy.get('[data-cy=role-item]').should('have.length', 3);
  });
});

describe('List roles', () => {
  it('Opens roles page', () => {
    cy.loginAndVisit('/admin/users/roles/lists');
  });

  it('Adds new roles', () => {
    cy.get('[data-cy=btn-new]').click();
    cy.get('[data-cy=role-form] #role-name').type('first');
    addList('Default list');
    saveRole();
    cy.contains('[data-cy=role-item]', 'first').should('exist');

    cy.get('[data-cy=btn-new]').click();
    cy.get('[data-cy=role-form] #role-name').type('second');
    addList('Default list');
    addList('Opt-in list');
    cy.get('[data-cy=role-form] tbody tr').should('have.length', 2);
    saveRole();
    cy.contains('[data-cy=role-item]', 'second').should('exist');
  });

  it('Edits role', () => {
    selectRole('second');

    // Uncheck "manage" permission on the second list.
    cy.get('[data-cy=role-form] tbody tr').eq(1).find('input[type=checkbox]').eq(1)
      .uncheck({ force: true });
    saveRole();
    cy.toast().should('exist');
  });

  it('Deletes role', () => {
    selectRole('second');
    cy.get('[data-cy=role-form] [data-cy=btn-clone]').click();
    cy.dialog().find('[data-cy=btn-save]').click();
    cy.get('[data-cy=role-item]').should('have.length', 3);

    cy.get('[data-cy=role-form] #role-name').should('not.have.value', 'second');
    cy.get('[data-cy=role-form] [data-cy=btn-delete]').click();
    cy.confirmAction();

    cy.get('[data-cy=role-item]').should('have.length', 2);
  });
});

describe('Users ', () => {
  it('Opens users page', () => {
    cy.loginAndVisit('/admin/users');
  });

  it('Adds new users', () => {
    ['first', 'second', 'third'].forEach((name) => {
      cy.get('[data-cy=btn-new]').click();
      cy.dialog().within(() => {
        cy.get('#user-username').type(name);
        cy.get('#user-name').type(name);
        cy.get('#user-email').type(`${name}@domain`);
        cy.get('#user-pw-login').check({ force: true });
        cy.get('#user-password').type(`${name}000000`);
        cy.get('#user-password2').type(`${name}000000`);
      });

      const role = name !== 'third' ? name : 'first';
      cy.pvSelect('#user-role', role);
      cy.pvSelect('#user-list-role', role);
      cy.dialog().find('[data-cy=btn-save]').click();
      cy.get('.ln-dialog-form').should('not.exist');
    });

    // admin + the three new users (API users aren't listed here).
    cy.rows().should('have.length', 4);
  });

  it('Edits user', () => {
    cy.rows().last().then((row) => rowMenu(row, 'Edit'));
    cy.dialog().find('#user-pw-login').uncheck({ force: true });
    cy.pvSelect('#user-role', 'second');
    cy.pvSelect('#user-list-role', 'second');
    cy.dialog().find('[data-cy=btn-save]').click();
    cy.get('.ln-dialog-form').should('not.exist');

    // Fetch the users API and verify the values that couldn't be verified on the table UI.
    cy.request(`${apiUrl}/api/users/4`).should((response) => {
      const { data } = response.body;

      expect(data.password_login).to.equal(false);
      expect(data.user_role.name).to.equal('second');
      expect(data.list_role.name).to.equal('second');
    });
  });

  it('Deletes a user', () => {
    cy.rows().last().then((row) => rowMenu(row, 'Delete'));
    cy.confirmAction();
    cy.rows().should('have.length', 3);
  });
});

describe('Login ', () => {
  // On Lists rows, edit is a row button and delete is in the row's popup menu.
  const assertCanManage = (row) => {
    cy.wrap(row).find('[data-cy=btn-edit]').should('exist');
    cy.wrap(row).find('button[aria-haspopup=true]').click();
    cy.get('.p-menu:visible .p-menu-item').contains('Delete').should('exist');
    // Toggle the popup menu closed again.
    cy.wrap(row).find('button[aria-haspopup=true]').click();
    cy.get('.p-menu:visible').should('not.exist');
  };

  it('Logs in as first', () => {
    login('first', 'first000000');

    // first=only default list.
    cy.rows().should('have.length', 1);
    cy.rows().first().find('.ln-row-title').should('contain', 'Default list');
    cy.get('[data-cy=btn-new]').should('not.exist');
    cy.rows().first().then(assertCanManage);
  });

  it('Logs in as second', () => {
    login('second', 'second000000');

    // second=all lists.
    cy.rows().should('have.length', 2);
    cy.rows().eq(0).then(assertCanManage);
    cy.rows().eq(1).then(assertCanManage);
  });
});
