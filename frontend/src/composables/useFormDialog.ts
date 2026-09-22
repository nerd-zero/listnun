import { ref, type Ref } from 'vue';

export interface UseFormDialogOptions<T> {
  /** Factory for a blank record when opening the "new" form. Defaults to `{}`. */
  newItem?: () => T;
}

// The "create/edit a record in a modal" state machine shared by the Lists,
// Templates, Users, and Subscribers views: a single dialog toggled between
// a blank "new" record and an existing "edit" record.
export function useFormDialog<T = Record<string, unknown>>(
  options: UseFormDialogOptions<T> = {},
) {
  const { newItem = () => ({} as T) } = options;

  const curItem = ref<T | null>(null) as Ref<T | null>;
  const isEditing = ref(false);
  const isFormVisible = ref(false);

  function showEditForm(item: T) {
    curItem.value = item;
    isEditing.value = true;
    isFormVisible.value = true;
  }

  function showNewForm() {
    curItem.value = newItem();
    isEditing.value = false;
    isFormVisible.value = true;
  }

  function closeForm() {
    isFormVisible.value = false;
  }

  return {
    curItem, isEditing, isFormVisible, showEditForm, showNewForm, closeForm,
  };
}
