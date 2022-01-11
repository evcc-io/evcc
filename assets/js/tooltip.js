import "v-tooltip/dist/v-tooltip.css";
import Vue from "vue";
import VTooltip from "v-tooltip";
VTooltip.options.offset = [0, 10];
VTooltip.options.themes.tooltip.triggers = ["click", "focus"];

Vue.use(VTooltip);
