export default function (tariff, plan, startTime, targetTime) {
  const result = [];

  const oneHour = 60 * 60 * 1000;
  for (let i = 0; i < 24; i++) {
    const startTime = new Date(startTime + oneHour * i);
    const startHour = startTime.getHours();
    const endTime = new Date(startTime.getTime());
    endTime.setHours(startHour + 1);
    const endHour = endTime.getHours();
    const day = startHour === 0 ? this.weekdayShort(startTime) : "";
    const toLate = targetTime.getTime() < startTime.getTime();
    const price = 2;
    const chargeHours = 1;
    //const price = this.ratePrice(start, end); //Math.round((hour < 5 ? 100 : 400) + Math.random() * 300);
    result.push({ day, price, startHour, endHour, chargeHours, toLate });
  }

  return result;
}
