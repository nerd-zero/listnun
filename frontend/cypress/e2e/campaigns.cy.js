const apiUrl = Cypress.env('apiUrl');
const headers = '[{"X-Custom": "Custom-Value"}]';

// Labels of the content formats in the editor's format Select.
const formatLabels = {
  richtext: 'Rich text', html: 'Raw HTML', markdown: 'Markdown', plain: 'Plain text', visual: 'Visual',
};

// The campaign ID from the editor's URL (/admin/campaigns/:id).
const campaignID = (pathname) => pathname.split('/').at(-1).replace(/#.*/, '');

// Replaces the content of the CodeMirror 6 editor inside `sel`. CodeMirror
// keeps its view on the content node (`cmView`), which is what
// EditorView.findFromDOM() reads in @codemirror/view 6.x.
const setCode = (sel, text) => {
  cy.get(`${sel} .cm-content`).then(($el) => {
    const { view } = $el[0].cmView.rootView;
    view.dispatch({ changes: { from: 0, to: view.state.doc.length, insert: text } });
  });
};

// Sets the TinyMCE (richtext) body and fires `change` so tinymce-vue
// updates the bound value.
const setRichtext = (html) => {
  cy.window().then((win) => {
    win.tinymce.editors[0].setContent(html);
    win.tinymce.editors[0].fire('change');
  });
};

// Switching the format of a campaign that has a body asks for confirmation
// in the global PrimeVue ConfirmDialog; an empty body switches directly.
const acceptFormatSwitch = () => {
  cy.wait(300);
  cy.get('body').then(($body) => {
    if ($body.find('.p-confirmdialog-accept-button:visible').length) {
      cy.get('.p-confirmdialog-accept-button:visible').click();
    }
  });
};

// Adds the first not-yet-selected list in the campaign's ListSelector.
const pickList = () => {
  cy.get('.list-selector .p-autocomplete-dropdown').click();
  cy.get('.p-autocomplete-overlay .p-autocomplete-option').first().click();
};

// Clicks an item in a table row's popup menu.
const rowMenu = (row, label) => {
  cy.wrap(row).find('[data-cy=btn-row-menu]').click();
  cy.get('.p-menu-overlay:visible').contains('.p-menu-item', label).click();
};

// The text of a campaign's rendered preview (the iframe is sandboxed and
// inaccessible to JS, so this goes through the API).
const previewText = (id) => cy.request(`${apiUrl}/api/campaigns/${id}/preview`)
  .then((resp) => resp.body.replace(/<[^>]+>/g, ' ').replace(/&nbsp;/g, ' ').replace(/\s+/g, ' ').trim());

describe('Campaigns', () => {
  it('Opens campaigns page', () => {
    cy.resetDB();
    cy.loginAndVisit('/admin/campaigns');
  });

  it('Counts campaigns', () => {
    cy.get('[data-cy=campaign-name]').should('have.length', 1);
  });

  it('Creates campaign', () => {
    cy.get('[data-cy=btn-new]').click();

    // Fill fields.
    cy.get('input[name=name]').clear().type('new-attach');
    cy.get('input[name=subject]').clear().type('new-subject');
    cy.get('input[name=from_email]').clear().type('new <from@email>');
    pickList();

    // Creating the campaign opens the Content step.
    cy.get('[data-cy=btn-continue]').click();
    cy.location('hash').should('eq', '#content');

    // Upload a file in the media picker and attach it.
    cy.get('[data-cy=btn-attach]').click();
    cy.dialog().find('input[type=file]').selectFile({
      contents: Cypress.Buffer.from('{"example": true}'),
      fileName: 'example.json',
      mimeType: 'application/json',
    }, { force: true });
    cy.dialog().contains('.media-card', 'example').find('.media-thumb').click();
    cy.get('[data-cy=media]').contains('example');

    // Save; this moves on to the Schedule step.
    cy.get('[data-cy=btn-next]').click();
    cy.location('hash').should('eq', '#schedule');

    // Re-open and check that the file still exists.
    cy.visit('/admin/campaigns');
    cy.get('[data-cy=campaign-name]').first().click();
    cy.get('[data-cy=step-content]').click();
    cy.get('[data-cy=media]').contains('example');

    // Start from the Review step.
    cy.get('[data-cy=step-review]').click();
    cy.get('[data-cy=btn-start]').click();
    cy.confirmAction();
    cy.location('pathname').should('match', /\/campaigns$/);
    cy.rows().eq(0).find('.ln-status').should(($tag) => {
      expect(['running', 'finished']).to.include($tag.attr('data-status'));
    });
  });

  it('Edits campaign', () => {
    cy.get('[data-cy=campaign-name][data-id=1]').click();

    // Fill fields.
    cy.get('input[name=name]').clear().type('new-name');
    cy.get('input[name=subject]').clear().type('new-subject');
    cy.get('input[name=from_email]').clear().type('new <from@email>');

    // Change the list.
    cy.get('.list-selector .list-tags .pi-times').click({ multiple: true });
    pickList();

    // Clear and redo tags.
    cy.get('[data-cy=tags] input').type('{backspace}new-tag{enter}');

    // Add custom headers (under Advanced).
    cy.get('[data-cy=btn-advanced]').click();
    cy.get('textarea[name=headers]').invoke('val', headers).trigger('input');

    // Save and go to the Content step.
    cy.get('[data-cy=btn-next]').click();
    cy.location('hash').should('eq', '#content');

    // Switch format to plain text.
    cy.pvSelect('[data-cy=check-format]', formatLabels.plain);
    acceptFormatSwitch();

    // Enter body value; save and go to the Schedule step.
    cy.get('textarea[name=content]').clear().type('new-content');
    cy.get('[data-cy=btn-next]').click();
    cy.location('hash').should('eq', '#schedule');

    // Send later: pick a day next month.
    cy.get('[data-cy=btn-send-later]').click();
    cy.get('#camp-send-at').click();
    cy.get('.p-datepicker-next-button:visible').click();
    cy.get('.p-datepicker-day-cell:not(.p-datepicker-other-month) .p-datepicker-day:not(.p-disabled):visible').first().click();
    cy.get('.camp-schedule .ln-section-title').click();

    // Save, then schedule from the Review step.
    cy.get('[data-cy=btn-next]').click();
    cy.location('hash').should('eq', '#review');
    cy.get('[data-cy=btn-schedule]').click();
    cy.confirmAction();
    cy.location('pathname').should('match', /\/campaigns$/);

    // Verify the changes.
    cy.request(`${apiUrl}/api/campaigns/1`).should((response) => {
      const { data } = response.body;
      expect(data.status).to.equal('scheduled');
      expect(data.name).to.equal('new-name');
      expect(data.subject).to.equal('new-subject');
      expect(data.content_type).to.equal('plain');
      expect(data.altbody).to.equal(null);
      expect(data.send_at).to.not.equal(null);
      expect(data.body).to.equal('new-content');

      expect(data.lists.length).to.equal(1);
      expect(data.lists[0].id).to.equal(1);
      expect(data.tags.length).to.equal(1);
      expect(data.tags[0]).to.equal('new-tag');
      expect(data.headers[0]['X-Custom']).to.equal('Custom-Value');
    });

    cy.get('.p-datatable-tbody .ln-status[data-status=scheduled]');
  });

  it('Unschedules campaign', () => {
    cy.get('[data-cy=campaign-name][data-id=1]').click();
    cy.get('[data-cy=step-review]').click();
    cy.get('[data-cy=btn-unschedule]').click();
    cy.confirmAction();
    cy.get('[data-cy=btn-unschedule]').should('not.exist');
    cy.visit('/admin/campaigns');

    // Only the edited campaign is a draft; the other one was started.
    cy.get('.p-datatable-tbody .ln-status[data-status=draft]').should('have.length', 1);
  });

  it('Switches formats', () => {
    cy.resetDB();
    cy.loginAndVisit('/admin/campaigns');
    const formats = ['html', 'markdown', 'plain'];
    const htmlBody = '<strong>hello</strong> {{ .Subscriber.Name }} from {{ .Subscriber.Attribs.city }}';
    const plainBody = 'hello Demo Subscriber from Bengaluru';

    // Set test content the first time.
    cy.get('[data-cy=campaign-name]').first().click();
    cy.get('[data-cy=step-content]').click();
    setRichtext(htmlBody);
    cy.get('[data-cy=btn-next]').click();
    cy.location('hash').should('eq', '#schedule');

    formats.forEach((c) => {
      cy.visit('/admin/campaigns');
      cy.get('[data-cy=campaign-name]').first().click();

      // Switch to the Content step and change the format.
      cy.get('[data-cy=step-content]').click();
      cy.pvSelect('[data-cy=check-format]', formatLabels[c]);
      acceptFormatSwitch();
      cy.wait(300);

      // Save the converted content.
      cy.get('[data-cy=btn-next]').click();
      cy.location('hash').should('eq', '#schedule');

      // Check the saved format and the rendered content.
      cy.location('pathname').then((p) => {
        cy.request(`${apiUrl}/api/campaigns/${campaignID(p)}`).its('body.data.content_type').should('eq', c);
        if (c !== 'plain') {
          previewText(campaignID(p)).should('contain', plainBody);
        }
      });
    });
  });

  it('Clones campaign', () => {
    cy.visit('/admin/campaigns');
    for (let n = 0; n < 3; n += 1) {
      // Clone the campaign; the clone opens in the editor.
      cy.get('[data-cy=btn-clone]').first().click();
      cy.get('#clone-name').clear().type(`clone${n}`);
      cy.dialog().find('[data-cy=btn-save]').click();
      cy.location('pathname').should('match', /\/campaigns\/\d+$/);
      cy.visit('/admin/campaigns');

      // Verify the newly created row.
      cy.get('[data-cy=campaign-name]').first().contains(`clone${n}`);
    }
  });

  it('Searches campaigns', () => {
    cy.get('input[name=query]').clear().type('clone2{enter}');
    cy.get('[data-cy=campaign-name]').should('have.length', 1);
    cy.get('[data-cy=campaign-name]').first().contains('clone2');
    cy.get('input[name=query]').clear().type('{enter}');
    cy.get('[data-cy=campaign-name]').should('have.length', 4);
  });

  it('Deletes campaigns', () => {
    // Delete all visible campaigns.
    cy.rows().each(() => {
      cy.rows().first().then((row) => rowMenu(row, 'Delete'));
      cy.confirmAction();
      cy.wait(250);
    });

    // Confirm deletion.
    cy.get('.p-datatable-empty-message');
  });

  it('Adds new campaigns', () => {
    const lists = [[1], [1, 2]];
    const cTypes = ['richtext', 'html', 'markdown', 'plain', 'visual'];

    let n = 0;
    cTypes.forEach((c) => {
      lists.forEach((l) => {
        // Click the 'new button'
        cy.get('[data-cy=btn-new]').click();
        cy.location('pathname').should('match', /\/campaigns\/new$/);

        // Fill fields.
        cy.get('input[name=name]').clear().type(`name${n}`);
        cy.get('input[name=subject]').clear().type(`subject${n}`);

        l.forEach(() => pickList());

        // Add tags.
        for (let i = 0; i < 3; i += 1) {
          cy.get('[data-cy=tags] input').type(`tag${i}{enter}`);
        }

        // Add headers (under Advanced).
        cy.get('[data-cy=btn-advanced]').click();
        cy.get('textarea[name=headers]').invoke('val', `[{"X-Header-${n}": "Value-${n}"}]`).trigger('input');

        // Hit 'Continue'.
        cy.get('[data-cy=btn-continue]').click();
        cy.location('hash').should('eq', '#content');

        // Verify the changes.
        const idx = n;
        cy.location('pathname').then((p) => {
          cy.request(`${apiUrl}/api/campaigns/${campaignID(p)}`).should((response) => {
            const { data } = response.body;
            expect(data.status).to.equal('draft');
            expect(data.name).to.equal(`name${idx}`);
            expect(data.subject).to.equal(`subject${idx}`);
            expect(data.content_type).to.equal('richtext');
            expect(data.altbody).to.equal(null);
            expect(data.send_at).to.equal(null);
            expect(data.headers[0][`X-Header-${idx}`]).to.equal(`Value-${idx}`);
          });
        });

        // Select content type.
        if (c !== 'richtext') {
          cy.pvSelect('[data-cy=check-format]', formatLabels[c]);
          acceptFormatSwitch();
        }

        // Insert content.
        const htmlBody = `<strong>hello${n}</strong> {{ .Subscriber.Name }} from {{ .Subscriber.Attribs.city }}`;
        const plainBody = `hello${n} Demo Subscriber from Bengaluru`;
        const markdownBody = `**hello${n}** Demo Subscriber from Bengaluru`;

        cy.log(`format = ${c}`);
        if (c === 'richtext') {
          setRichtext(htmlBody);
          cy.wait(500);
        } else if (c === 'html') {
          setCode('.camp-content', htmlBody);
        } else if (c === 'markdown') {
          setCode('.camp-content', markdownBody);
        } else if (c === 'plain') {
          cy.get('textarea[name=content]').invoke('val', plainBody).trigger('input');
        } else if (c === 'visual') {
          cy.wait(200);
          cy.get('.camp-content iframe').then((el) => {
            cy.wait(200);
            cy.wrap(el.contents()).find('table td').click();
            cy.wait(200);
            cy.wrap(el.contents()).find('textarea').eq(0).type(plainBody);
          });
        }

        // Save; this moves on to the Schedule step.
        cy.get('[data-cy=btn-next]').click();
        cy.location('hash').should('eq', '#schedule');

        // Verify the preview content via the API.
        if (c !== 'plain') {
          cy.location('pathname').then((p) => {
            previewText(campaignID(p)).should('contain', plainBody);
          });
        }

        cy.visit('/admin/campaigns');

        // Verify the newly created campaign in the table. The row's
        // sub-line names the campaign's lists, comma-separated.
        cy.get('[data-cy=campaign-name]').first().contains(`name${n}`);
        cy.rows().first().find('.ln-row-sub').invoke('text')
          .then((t) => {
            expect(t.trim().split(', ').length).to.equal(l.length);
          });

        n += 1;
      });
    });

    // Fetch the campaigns API and verfiy the values that couldn't be verified on the table UI.
    cy.request(`${apiUrl}/api/campaigns?order=asc&order_by=created_at`).should((response) => {
      const { data } = response.body;
      expect(data.total).to.equal(lists.length * cTypes.length);

      let i = 0;
      cTypes.forEach((c) => {
        lists.forEach((l) => {
          expect(data.results[i].subject).to.equal(`subject${i}`);
          expect(data.results[i].content_type).to.equal(c);
          expect(data.results[i].lists.map((ls) => ls.id)).to.deep.equal(l);
          i += 1;
        });
      });
    });
  });

  it('Starts and cancels campaigns', () => {
    for (let n = 1; n <= 2; n += 1) {
      const row = `.p-datatable-tbody > tr:nth-child(${n})`;
      cy.get(row).then((el) => rowMenu(el, 'Start campaign'));
      cy.confirmAction();
      cy.get(`${row} .ln-status[data-status=running]`);

      if (n > 1) {
        cy.get(row).then((el) => rowMenu(el, 'Cancel'));
        cy.confirmAction();
        cy.get(`${row} .ln-status[data-status=cancelled]`);
      }
    }
  });

  it('Sorts campaigns', () => {
    const asc = [5, 6, 7, 8, 9, 10, 11, 12, 13, 14];
    const desc = [14, 13, 12, 11, 10, 9, 8, 7, 6, 5];
    const cases = ['cy-name', 'cy-timestamp'];

    cases.forEach((c) => {
      cy.sortTable(`thead th.${c}`, asc);
      cy.wait(250);
      cy.sortTable(`thead th.${c}`, desc);
      cy.wait(250);
    });
  });

  it('Bulk deletes campaigns', () => {
    const params = {
      name: 'test-campaign-bulk',
      subject: 'subject',
      type: 'regular',
      content_type: 'richtext',
      template_id: 1,
      lists: [1],
    };

    // Create 30 in a loop.
    for (let i = 0; i < 30; i += 1) {
      cy.request('POST', `${apiUrl}/api/campaigns`, params);
    }

    cy.visit('/admin/campaigns');

    // Bulk delete with the `all` flag, which asks for the count to be typed.
    cy.window().scrollTo('top');
    cy.get('[data-cy=campaign-name]').should('have.length.at.least', 1);
    cy.get('thead input[type="checkbox"]').click({ force: true });
    cy.get('[data-cy=select-all-campaigns]').click();
    cy.get('[data-cy=btn-delete-campaigns]').click();
    cy.get('.ln-confirm:visible code').invoke('text').then((num) => cy.confirmAction(num.trim()));
    cy.get('.p-datatable-empty-message');

    // Bulk delete with the selected IDs.
    // Create 5 campaigns in a loop.
    for (let i = 0; i < 5; i += 1) {
      cy.request('POST', `${apiUrl}/api/campaigns`, params);
    }

    cy.visit('/admin/campaigns');
    cy.get('[data-cy=campaign-name]').should('have.length', 5);
    cy.get('thead input[type="checkbox"]').click({ force: true });
    cy.get('[data-cy=btn-delete-campaigns]').click();
    cy.confirmAction();
    cy.get('.p-datatable-empty-message');
  });
});
