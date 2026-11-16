import { reactive } from 'vue';
import { readJSON, writeJSON } from '../safeStorage';

// Nav groups the user opened or closed by hand. Kept at module level so the
// sidebar and the drawer share it and it survives the sidebar/rail swap,
// and saved so it sticks across reloads.
const STORAGE_KEY = 'ln-nav-open';

type OpenState = Record<string, boolean>;

function load(): OpenState {
  const v = readJSON(STORAGE_KEY);
  if (!v || typeof v !== 'object' || Array.isArray(v)) return {};
  return Object.fromEntries(Object.entries(v).filter(([, open]) => typeof open === 'boolean')) as OpenState;
}

const opened = reactive<OpenState>(load());

// If storage is blocked, the state still holds for this visit.
function save() {
  writeJSON(STORAGE_KEY, opened);
}

export function useNavOpen() {
  // A group the user never touched is open while one of its pages is current.
  function isOpen(key: string, active: boolean): boolean {
    return opened[key] ?? active;
  }

  function setOpen(key: string, open: boolean) {
    opened[key] = open;
    save();
  }

  return { isOpen, setOpen };
}

export default useNavOpen;
