import type { Meta, StoryFn } from "@storybook/vue3";
import { ICON_SIZE } from "@/types/evcc";

// Auto-discover all MaterialIcon components
const iconModules = import.meta.glob("./*.vue", { eager: true });
const iconComponents: Record<string, any> = {};
const iconNames: string[] = [];

Object.keys(iconModules).forEach((path) => {
  const iconName = path.replace("./", "").replace(".vue", "");
  iconComponents[iconName] = (iconModules[path] as any).default;
  iconNames.push(iconName);
});

// Sort icon names alphabetically
iconNames.sort();

export default {
  title: "MaterialIcon/MaterialIcon",
  argTypes: {
    size: {
      control: "select",
      options: Object.values(ICON_SIZE),
      defaultValue: ICON_SIZE.L,
    },
  },
} as Meta;

// Template for a single icon
const SingleIconTemplate: StoryFn = (args) => ({
  components: iconComponents,
  setup() {
    return { args, iconNames };
  },
  template: `
    <div>
      <component :is="args.iconName" v-bind="args" />
    </div>
  `,
});

// Single icon story (controllable via Storybook controls)
export const SingleIcon = SingleIconTemplate.bind({});
SingleIcon.args = {
  iconName: iconNames[0],
  size: ICON_SIZE.L,
};
SingleIcon.argTypes = {
  iconName: {
    control: "select",
    options: iconNames,
  },
};

// All icons in a simple grid
export const AllIcons = () => ({
  components: iconComponents,
  setup() {
    return { iconNames };
  },
  template: `
    <div style="display: grid; grid-template-columns: repeat(auto-fill, minmax(120px, 1fr)); gap: 20px;">
        <div v-for="iconName in iconNames" :key="iconName" style="display: flex; flex-direction: column; align-items: center; gap: 10px; padding: 15px;">
            <component :is="iconName" size="xl" />
            <small style="font-family: monospace; color: #666; text-align: center; font-size: 12px;">{{ iconName }}</small>
        </div>
    </div>
  `,
});

// Color variations story - table format with all icons
export const ColorVariations = () => ({
  components: iconComponents,
  setup() {
    return { iconNames };
  },
  template: `
      <table style="width: 100%; border-collapse: collapse; font-size: 14px;">
        <thead>
          <tr style="border-bottom: 2px solid #ddd;">
            <th style="padding: 12px; text-align: left; font-weight: normal;">Icon Name</th>
            <th style="padding: 12px; text-align: left; font-weight: normal;">Default</th>
            <th style="padding: 12px; text-align: left; font-weight: normal; color: var(--bs-primary);">Primary</th>
            <th style="padding: 12px; text-align: left; font-weight: normal; color: var(--bs-danger);">Danger</th>
            <th style="padding: 12px; text-align: left; font-weight: normal; color: var(--bs-warning);">Warning</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="iconName in iconNames" :key="iconName" style="border-bottom: 1px solid #eee;">
            <td style="padding: 12px; font-family: monospace; color: #666;">{{ iconName }}</td>
            <td style="padding: 12px;">
              <component :is="iconName" size="l" />
            </td>
            <td style="padding: 12px;">
              <component :is="iconName" size="l" class="text-primary" />
            </td>
            <td style="padding: 12px;">
              <component :is="iconName" size="l" class="text-danger" />
            </td>
            <td style="padding: 12px;">
              <component :is="iconName" size="l" class="text-warning" />
            </td>
          </tr>
        </tbody>
      </table>
  `,
});

// Size variations story - all icons
export const SizeVariations = () => ({
  components: iconComponents,
  setup() {
    const sizes = Object.values(ICON_SIZE);
    return { iconNames, sizes };
  },
  template: `
      <div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(350px, 1fr)); gap: 30px;">
        <div v-for="iconName in iconNames" :key="iconName" style="display: flex; flex-direction: column; align-items: center; gap: 20px; padding: 20px; border: 1px solid #e0e0e0; border-radius: 8px;">
          <h3 style="margin: 0; font-size: 16px; color: #333; font-family: monospace;">{{ iconName }}</h3>
          <div style="display: flex; align-items: end; gap: 20px;">
            <div v-for="size in sizes" :key="size" style="display: flex; flex-direction: column; align-items: center; gap: 8px;">
              <component :is="iconName" :size="size" />
              <small style="color: #999; font-size: 11px; font-family: monospace;">{{ size }}</small>
            </div>
          </div>
        </div>
      </div>
  `,
});
