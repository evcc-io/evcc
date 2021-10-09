import confetti from "canvas-confetti";

export default {
  methods: {
    confetti($el, direction = "upLeft") {
      const directions = {
        upLeft: { angle: 90 + Math.random() * 35, drift: -0.5 },
        up: { angle: 45 + Math.random() * 90, drift: 0 },
      };
      const { angle, drift } = directions[direction];

      const { top, height, left, width } = $el.getBoundingClientRect();
      const x = (left + width / 2) / window.innerWidth;
      const y = (top + height / 2) / window.innerHeight;
      const origin = { x, y };

      confetti({
        origin,
        angle,
        particleCount: 75 + Math.random() * 50,
        spread: 50 + Math.random() * 50,
        drift,
        scalar: 1.3,
        zIndex: 1056, // Bootstrap Modal is 1055
        colors: [
          "#0d6efd",
          "#0fdd42",
          "#408458",
          "#4923BA",
          "#5BC8EC",
          "#C54482",
          "#CC444A",
          "#EE8437",
          "#F7C144",
          "#FFFD54",
        ],
      });
    },
  },
};
