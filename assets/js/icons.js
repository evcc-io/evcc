import Vue from "vue";
import { FontAwesomeIcon } from "@fortawesome/vue-fontawesome";
import { library } from "@fortawesome/fontawesome-svg-core";

import { faArrowDown } from "@fortawesome/free-solid-svg-icons/faArrowDown";
import { faArrowUp } from "@fortawesome/free-solid-svg-icons/faArrowUp";
import { faBatteryEmpty } from "@fortawesome/free-solid-svg-icons/faBatteryEmpty";
import { faBatteryFull } from "@fortawesome/free-solid-svg-icons/faBatteryFull";
import { faBatteryHalf } from "@fortawesome/free-solid-svg-icons/faBatteryHalf";
import { faBatteryQuarter } from "@fortawesome/free-solid-svg-icons/faBatteryQuarter";
import { faBatteryThreeQuarters } from "@fortawesome/free-solid-svg-icons/faBatteryThreeQuarters";
import { faChevronDown } from "@fortawesome/free-solid-svg-icons/faChevronDown";
import { faChevronUp } from "@fortawesome/free-solid-svg-icons/faChevronUp";
import { faClock } from "@fortawesome/free-solid-svg-icons";
import { faExclamationTriangle } from "@fortawesome/free-solid-svg-icons/faExclamationTriangle";
import { faLeaf } from "@fortawesome/free-solid-svg-icons/faLeaf";
import { faSun } from "@fortawesome/free-solid-svg-icons/faSun";
import { faTemperatureHigh } from "@fortawesome/free-solid-svg-icons/faTemperatureHigh";
import { faTemperatureLow } from "@fortawesome/free-solid-svg-icons/faTemperatureLow";
import { faThermometerHalf } from "@fortawesome/free-solid-svg-icons/faThermometerHalf";
import { faHeart } from "@fortawesome/free-solid-svg-icons/faHeart";
import { faGift } from "@fortawesome/free-solid-svg-icons/faGift";
import { faBox } from "@fortawesome/free-solid-svg-icons/faBox";

library.add(
  faArrowDown,
  faArrowUp,
  faBatteryEmpty,
  faBatteryFull,
  faBatteryHalf,
  faBatteryQuarter,
  faBatteryThreeQuarters,
  faChevronDown,
  faChevronUp,
  faClock,
  faExclamationTriangle,
  faLeaf,
  faSun,
  faTemperatureHigh,
  faTemperatureLow,
  faThermometerHalf,
  faHeart,
  faGift,
  faBox
);

Vue.component("fa-icon", FontAwesomeIcon);
