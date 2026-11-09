// A signup form as the forms API returns it (camelCased).
export interface FormField {
  // email, name, or attribs.<name>.
  key: string;
  label: string;
  required: boolean;
}

export interface SignupForm {
  id: number;
  uuid: string;
  slug: string;
  name: string;
  status: 'live' | 'draft';
  heading: string;
  intro: string;
  listIds: number[];
  fields: FormField[];
  buttonLabel: string;
  successMessage: string;
  captcha: boolean;
  signups: number;
  url: string;
  embedHtml: string;
  popupScript: string;
}

// The part of a form the editor changes.
export type FormDraft = Pick<SignupForm, 'name' | 'status' | 'heading' | 'intro' | 'listIds' | 'fields'
  | 'buttonLabel' | 'successMessage' | 'captcha'>;

export interface PublicList {
  id: number;
  uuid: string;
  name: string;
  optin: string;
  description?: string;
}

export const ATTRIB_PREFIX = 'attribs.';

// Turns a label into an attribute name the server accepts.
export function attribKey(label: string): string {
  return label.toLowerCase().replace(/[^a-z0-9]+/g, '_').replace(/^_+|_+$/g, '').slice(0, 64);
}

export function toDraft(f: SignupForm): FormDraft {
  return {
    name: f.name,
    status: f.status,
    heading: f.heading,
    intro: f.intro,
    listIds: [...(f.listIds || [])],
    fields: (f.fields || []).map((x) => ({ ...x })),
    buttonLabel: f.buttonLabel,
    successMessage: f.successMessage,
    captcha: f.captcha,
  };
}

// The request body for saving a draft.
export function toReq(d: FormDraft) {
  return {
    name: d.name,
    status: d.status,
    heading: d.heading,
    intro: d.intro,
    list_ids: d.listIds,
    fields: d.fields,
    button_label: d.buttonLabel,
    success_message: d.successMessage,
    captcha: d.captcha,
  };
}
