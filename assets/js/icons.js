import Vue from "vue";
import { FontAwesomeIcon } from "@fortawesome/vue-fontawesome";
import { library } from "@fortawesome/fontawesome-svg-core";
import { faSun } from "@fortawesome/free-solid-svg-icons/faSun";
import { faArrowUp } from "@fortawesome/free-solid-svg-icons/faArrowUp";
import { faArrowDown } from "@fortawesome/free-solid-svg-icons/faArrowDown";
import { faBatteryThreeQuarters } from "@fortawesome/free-solid-svg-icons/faBatteryThreeQuarters";
import { faTemperatureLow } from "@fortawesome/free-solid-svg-icons/faTemperatureLow";
import { faTemperatureHigh } from "@fortawesome/free-solid-svg-icons/faTemperatureHigh";
import { faThermometerHalf } from "@fortawesome/free-solid-svg-icons/faThermometerHalf";
import { faLeaf } from "@fortawesome/free-solid-svg-icons/faLeaf";
import { faChevronUp } from "@fortawesome/free-solid-svg-icons/faChevronUp";
import { faChevronDown } from "@fortawesome/free-solid-svg-icons/faChevronDown";
import { faExclamationTriangle } from "@fortawesome/free-solid-svg-icons/faExclamationTriangle";

library.add(
  faSun,
  faArrowUp,
  faArrowDown,
  faBatteryThreeQuarters,
  faTemperatureLow,
  faTemperatureHigh,
  faThermometerHalf,
  faLeaf,
  faChevronUp,
  faChevronDown,
  faExclamationTriangle
);

Vue.component("fa-icon", FontAwesomeIcon);
