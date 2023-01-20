import { mount, config } from "@vue/test-utils";
import { beforeAll, describe, expect, test } from "vitest";
import TargetChargePlan2 from "./TargetChargePlan2.vue";

config.global.mocks["$i18n"] = { locale: "de-DE" };

describe("basics", () => {
  const DATE_START = new Date("11 Jan 2023 11:00:00 GMT+001");
  const DATE_TARGET = new Date("11 Jan 2023 13:00:00 GMT+001");
  const TARIFF_FIXED = [
    {
      start: "2023-01-11T11:00:00+01:00",
      end: "2023-01-22T00:00:00+01:00",
      price: 0.4,
    },
  ];
  const PLAN = [
    {
      start: "11 Jan 2023 12:00:00 GMT+001",
      end: "11 Jan 2023 13:00:00 GMT+001",
    },
  ];

  const wrapper = mount(TargetChargePlan2, {
    props: {
      plan: PLAN,
      targetTime: DATE_TARGET,
      rates: TARIFF_FIXED,
    },
  });
  wrapper.setData({ startTime: DATE_START });

  let result = null;

  beforeAll(() => {
    result = wrapper.vm.slots;
  });

  test("should return 36 slots", () => {
    expect(result.length).eq(36);
  });

  test("slots should be an hour apart", () => {
    expect(result[0].startHour).eq(11);
    expect(result[0].endHour).eq(12);
    expect(result[0].day).eq("Mi");

    expect(result[1].startHour).eq(12);
    expect(result[1].endHour).eq(13);
    expect(result[1].day).eq("Mi");

    expect(result[11].startHour).eq(22);
    expect(result[11].endHour).eq(23);
    expect(result[11].day).eq("Mi");

    expect(result[24].startHour).eq(11);
    expect(result[24].endHour).eq(12);
    expect(result[24].day).eq("Do");
  });

  test("slots after target should be toLate", () => {
    expect(result[0].toLate).eq(false);
    expect(result[1].toLate).eq(false);
    expect(result[2].toLate).eq(true);
    expect(result[3].toLate).eq(true);
  });

  test("slots are marked if charging is happening in them", () => {
    expect(result[0].charging).eq(false);
    expect(result[1].charging).eq(true);
    expect(result[2].charging).eq(false);
    expect(result[3].charging).eq(false);
  });

  test("all slots have the same fixed price", () => {
    result.forEach((slot) => expect(slot.price).eq(0.4));
  });
});

describe("zoned tariffs", () => {
  const DATE_START = new Date("11 Jan 2023 11:00:00 GMT+001");
  const DATE_TARGET = new Date("12 Jan 2023 11:00:00 GMT+001");
  const TARIFF_ZONED = [
    {
      start: "2023-01-11T11:00:00+01:00",
      end: "2023-01-11T12:00:00+01:00",
      price: 0.2,
    },
    {
      start: "2023-01-11T12:00:00+01:00",
      end: "2023-01-22T00:00:00+01:00",
      price: 0.4,
    },
  ];
  const PLAN = [
    {
      start: "11 Jan 2023 11:30:00 GMT+001",
      end: "11 Jan 2023 13:00:00 GMT+001",
    },
    {
      start: "12 Jan 2023 9:30:00 GMT+001",
      end: "12 Jan 2023 11:00:00 GMT+001",
    },
  ];

  let result = null;

  const wrapper = mount(TargetChargePlan2, {
    props: {
      plan: PLAN,
      targetTime: DATE_TARGET,
      rates: TARIFF_ZONED,
    },
  });
  wrapper.setData({ startTime: DATE_START });

  beforeAll(() => {
    result = wrapper.vm.slots;
  });

  test("handle multiple charging slots", () => {
    result.forEach((slot, index) => {
      expect(slot.charging).eq([0, 1, 22, 23].includes(index));
    });
  });

  test("first slot is cheap, others are expensive", () => {
    const [first, ...others] = result;
    expect(first.price).eq(0.2);
    others.forEach((slot) => expect(slot.price).eq(0.4));
  });
});
