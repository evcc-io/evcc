import Vue from "vue";
import { FontAwesomeIcon } from "@fortawesome/vue-fontawesome";
import { library } from "@fortawesome/fontawesome-svg-core";

import { faBatteryEmpty } from "@fortawesome/free-solid-svg-icons/faBatteryEmpty";
import { faBatteryFull } from "@fortawesome/free-solid-svg-icons/faBatteryFull";
import { faBatteryHalf } from "@fortawesome/free-solid-svg-icons/faBatteryHalf";
import { faBatteryQuarter } from "@fortawesome/free-solid-svg-icons/faBatteryQuarter";
import { faBatteryThreeQuarters } from "@fortawesome/free-solid-svg-icons/faBatteryThreeQuarters";
import { faAngleUp } from "@fortawesome/free-solid-svg-icons/faAngleUp";
import { faAngleDown } from "@fortawesome/free-solid-svg-icons/faAngleDown";
import { faClock } from "@fortawesome/free-solid-svg-icons/faClock";
import { faExclamationTriangle } from "@fortawesome/free-solid-svg-icons/faExclamationTriangle";
import { faSun } from "@fortawesome/free-solid-svg-icons/faSun";
import { faFlask } from "@fortawesome/free-solid-svg-icons/faFlask";
import { faTemperatureHigh } from "@fortawesome/free-solid-svg-icons/faTemperatureHigh";
import { faTemperatureLow } from "@fortawesome/free-solid-svg-icons/faTemperatureLow";
import { faThermometerHalf } from "@fortawesome/free-solid-svg-icons/faThermometerHalf";
import { faHeart as farHeart } from "@fortawesome/free-regular-svg-icons/faHeart";
import { faHeart as fasHeart } from "@fortawesome/free-solid-svg-icons/faHeart";
import { faGift } from "@fortawesome/free-solid-svg-icons/faGift";
import { faBox } from "@fortawesome/free-solid-svg-icons/faBox";
import { faHome } from "@fortawesome/free-solid-svg-icons/faHome";
import { faCar } from "@fortawesome/free-solid-svg-icons/faCar";
import { faEllipsisH } from "@fortawesome/free-solid-svg-icons/faEllipsisH";
import { faEllipsisV } from "@fortawesome/free-solid-svg-icons/faEllipsisV";
import { faSquare } from "@fortawesome/free-solid-svg-icons/faSquare";
import { faExclamationCircle } from "@fortawesome/free-solid-svg-icons/faExclamationCircle";
import { faAngleDoubleLeft } from "@fortawesome/free-solid-svg-icons/faAngleDoubleLeft";
import { faAngleDoubleRight } from "@fortawesome/free-solid-svg-icons/faAngleDoubleRight";

library.add(
  faAngleDown,
  faAngleUp,
  faBatteryEmpty,
  faBatteryFull,
  faBatteryHalf,
  faBatteryQuarter,
  faBatteryThreeQuarters,
  faBox,
  faCar,
  faAngleDoubleLeft,
  faAngleDoubleRight,
  faClock,
  faExclamationCircle,
  faExclamationTriangle,
  faGift,
  faHome,
  farHeart,
  fasHeart,
  faSquare,
  faSun,
  faFlask,
  faTemperatureHigh,
  faTemperatureLow,
  faThermometerHalf,
  faEllipsisH,
  faEllipsisV
);

// eslint-disable-next-line vue/component-definition-name-casing
Vue.component("fa-icon", FontAwesomeIcon);
