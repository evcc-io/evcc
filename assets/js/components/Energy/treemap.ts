export interface TreemapRect {
  x: number;
  y: number;
  width: number;
  height: number;
}

// Squarified treemap (Bruls et al.). Values must be sorted descending.
// `aspect` stretches the target shape: 2 prefers tiles twice as wide as high.
export function squarify(
  values: number[],
  width: number,
  height: number,
  aspect = 1
): TreemapRect[] {
  const total = values.reduce((a, b) => a + b, 0);
  const out: TreemapRect[] = [];
  if (!total || width <= 0 || height <= 0)
    return values.map(() => ({ x: 0, y: 0, width: 0, height: 0 }));

  // lay out in a space where the preferred tile is square, scale back at the end
  const w = width / aspect;
  const scale = (w * height) / total;
  let x = 0;
  let y = 0;
  let free = { w, h: height };
  let row: number[] = [];

  const worst = (r: number[], side: number) => {
    const sum = r.reduce((a, b) => a + b, 0) * scale;
    const max = Math.max(...r) * scale;
    const min = Math.min(...r) * scale;
    const s2 = side * side;
    return Math.max((s2 * max) / (sum * sum), (sum * sum) / (s2 * min));
  };

  const flush = () => {
    const sum = row.reduce((a, b) => a + b, 0) * scale;
    const horizontal = free.w >= free.h; // row fills the shorter side
    const thickness = sum / (horizontal ? free.h : free.w);
    let offset = 0;
    for (const v of row) {
      const len = (v * scale) / thickness;
      out.push(
        horizontal
          ? { x, y: y + offset, width: thickness, height: len }
          : { x: x + offset, y, width: len, height: thickness }
      );
      offset += len;
    }
    if (horizontal) {
      x += thickness;
      free = { w: free.w - thickness, h: free.h };
    } else {
      y += thickness;
      free = { w: free.w, h: free.h - thickness };
    }
    row = [];
  };

  for (const v of values) {
    const side = Math.min(free.w, free.h);
    if (row.length && worst([...row, v], side) > worst(row, side)) flush();
    row.push(v);
  }
  if (row.length) flush();

  return out.map((r) => ({ ...r, x: r.x * aspect, width: r.width * aspect }));
}
