<template>
  <nav :aria-label="$t('menu.main')" class="ln-nav" :class="{ 'ln-nav--rail': rail }">
    <template v-if="rail">
      <router-link
        v-for="item in railItems"
        :key="item.key"
        v-tooltip.right="item.label"
        :to="item.to"
        class="ln-rail-item"
        :class="{ 'is-active': isItemActive(item) }"
        :aria-label="item.label"
        :data-cy="item.cy || item.key"
      >
        <ln-icon :name="item.icon" />
      </router-link>
    </template>

    <template v-else>
      <div v-for="group in groups" :key="group.key" class="ln-nav-group">
        <span v-if="group.label" class="ln-nav-label">{{ group.label }}</span>

        <template v-for="item in group.items" :key="item.key">
          <button
            v-if="item.kids"
            type="button"
            class="ln-nav-item"
            :class="{ 'is-current': isItemActive(item) }"
            :aria-expanded="isOpen(item)"
            :data-cy="item.cy || item.key"
            @click="toggle(item)"
          >
            <ln-icon :name="item.icon" class="ln-nav-icon" />
            <span class="ln-nav-text">{{ item.label }}</span>
            <ln-icon name="chevronRight" :size="12" :stroke="1.6" class="ln-nav-chevron" />
          </button>
          <router-link
            v-else
            :to="item.to"
            class="ln-nav-item"
            :class="{ 'is-active': isItemActive(item) }"
            :aria-current="isItemActive(item) ? 'page' : undefined"
            :data-cy="item.cy || item.key"
          >
            <ln-icon :name="item.icon" class="ln-nav-icon" />
            <span class="ln-nav-text">{{ item.label }}</span>
          </router-link>

          <!-- Always rendered so opening and closing can animate; inert
               keeps the hidden links out of the tab order. -->
          <div
            v-if="item.kids"
            class="ln-nav-kids-wrap"
            :class="{ 'is-open': isOpen(item) }"
            :inert="!isOpen(item) || undefined"
          >
            <div class="ln-nav-kids">
              <router-link
                v-for="k in item.kids"
                :key="k.key"
                :to="k.to"
                class="ln-nav-kid"
                :class="{ 'is-active': isKidActive(k) }"
                :aria-current="isKidActive(k) ? 'page' : undefined"
                :data-cy="k.cy || k.key"
              >
                {{ k.label }}
              </router-link>
            </div>
          </div>
        </template>
      </div>
    </template>
  </nav>
</template>

<script setup lang="ts">
import { computed, reactive } from 'vue';
import { useRoute } from 'vue-router';
import { useI18n } from 'vue-i18n';
import { useGlobal } from '../composables/useGlobal';
import { navGroups, type NavItem, type NavKid } from './navigation';
import LnIcon from './ui/LnIcon.vue';

withDefaults(defineProps<{ rail?: boolean }>(), { rail: false });

const { t } = useI18n();
const route = useRoute();
const { $can } = useGlobal();

// Groups the user opened or closed by hand; otherwise a group is open
// while one of its pages is current.
const opened = reactive<Record<string, boolean>>({});

const groups = computed(() => navGroups(t, $can));

// The rail shows top-level destinations only; groups link to their first page.
// The Developers group collapses to a single entry.
const railItems = computed<NavItem[]>(() => groups.value.flatMap((g) => {
  if (g.key === 'developers') {
    return [{
      key: 'developers', label: g.label, icon: 'code', to: g.items[0].to, group: 'developers',
    }];
  }
  return g.items.map((item) => ({
    ...item,
    to: item.kids?.length ? item.kids[0].to : item.to,
  }));
}));

function isItemActive(item: NavItem): boolean {
  if (item.names?.includes(route.name as string)) return true;
  return !!item.group && route.meta.group === item.group;
}

function isKidActive(k: NavKid): boolean {
  return k.names.includes(route.name as string);
}

function isOpen(item: NavItem): boolean {
  return opened[item.key] ?? isItemActive(item);
}

function toggle(item: NavItem) {
  opened[item.key] = !isOpen(item);
}
</script>
