const board = document.getElementById("ludo-board");

/*
  15×15 Ludo layout mask
  H = home
  P = path
  S = safe path
  C = center
  . = empty
*/

const layout = [
  "HHHHH.....YYYYY",
  "HHHHH.....YYYYY",
  "HHHHH.....YYYYY",
  "HHHHH.....YYYYY",
  "HHHHH.....YYYYY",

  ".....PPPSPPP....",
  ".....PPPSPPP....",
  ".....PPPCPPP....",
  ".....PPPSPPP....",
  ".....PPPSPPP....",

  "BBBBB.....RRRRR",
  "BBBBB.....RRRRR",
  "BBBBB.....RRRRR",
  "BBBBB.....RRRRR",
  "BBBBB.....RRRRR"
];

function createCell(type, row, col) {
  const div = document.createElement("div");
  div.classList.add("cell");

  // Home zones
  if (type === "H") {
    if (row < 5) div.classList.add("green-home");
    else if (col < 5) div.classList.add("blue-home");
    else if (col > 9) div.classList.add("red-home");
    else div.classList.add("yellow-home");
  }

  // Path
  if (type === "P") div.classList.add("path");

  // Safe path
  if (type === "S") {
    div.classList.add("path", "safe");
  }

  // Center
  if (type === "C") div.classList.add("center");

  // Click event
  div.dataset.row = row;
  div.dataset.col = col;

  div.addEventListener("click", () => {
    console.log(`Clicked cell: ${row}, ${col}`);
  });

  return div;
}

layout.forEach((rowStr, r) => {
  [...rowStr].forEach((char, c) => {
    const cell = createCell(char, r, c);
    board.appendChild(cell);
  });
});
