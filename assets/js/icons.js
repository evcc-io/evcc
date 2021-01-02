import Vue from "vue";
import { FontAwesomeIcon } from "@fortawesome/vue-fontawesome";
import { library } from "@fortawesome/fontawesome-svg-core";
import { faArrowDown } from "@fortawesome/free-solid-svg-icons/faArrowDown";
import { faArrowUp } from "@fortawesome/free-solid-svg-icons/faArrowUp";
import { faBatteryThreeQuarters } from "@fortawesome/free-solid-svg-icons/faBatteryThreeQuarters";
import { faChevronUp } from "@fortawesome/free-solid-svg-icons/faChevronUp";
import { faChevronDown } from "@fortawesome/free-solid-svg-icons/faChevronDown";
import { faClock } from "@fortawesome/free-solid-svg-icons";
import { faExclamationTriangle } from "@fortawesome/free-solid-svg-icons/faExclamationTriangle";
import { faLeaf } from "@fortawesome/free-solid-svg-icons/faLeaf";
import { faSun } from "@fortawesome/free-solid-svg-icons/faSun";
import { faTemperatureLow } from "@fortawesome/free-solid-svg-icons/faTemperatureLow";
import { faTemperatureHigh } from "@fortawesome/free-solid-svg-icons/faTemperatureHigh";
import { faThermometerHalf } from "@fortawesome/free-solid-svg-icons/faThermometerHalf";

library.add(
  faArrowDown,
  faArrowUp,
  faBatteryThreeQuarters,
  faChevronDown,
  faChevronUp,
  faClock,
  faExclamationTriangle,
  faLeaf,
  faSun,
  faTemperatureLow,
  faTemperatureHigh,
  faThermometerHalf
);

Vue.component("fa-icon", FontAwesomeIcon);
