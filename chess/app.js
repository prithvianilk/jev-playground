import { Chess } from "https://cdn.jsdelivr.net/npm/chess.js@1.4.0/+esm";

const icons = { p: "♟", n: "♞", b: "♝", r: "♜", q: "♛", k: "♚" };
const names = { p: "PAWN", n: "KNIGHT", b: "BISHOP", r: "ROOK", q: "QUEEN", k: "KING" };
const chess = new Chess();
const board = document.querySelector("#board");
const status = document.querySelector("#status");
const hint = document.querySelector("#hint");
const movesElement = document.querySelector("#moves");
const moveCount = document.querySelector("#move-count");
let selected = null;
let thinking = false;

function boardState() {
  return chess.board().map(row => row.map(piece => piece ? `${piece.color === "w" ? "WHITE" : "BLACK"},${names[piece.type]}` : "EMPTY").join("|")).join("\n");
}

function render() {
  board.replaceChildren();
  chess.board().forEach((row, rank) => row.forEach((piece, file) => {
    const square = `${String.fromCharCode(97 + file)}${8 - rank}`;
    const element = document.createElement("button");
    element.className = `square ${(rank + file) % 2 ? "dark" : "light"}`;
    element.dataset.square = square;
    if (selected === square) element.classList.add("selected");
    if (selected && chess.moves({ square: selected, verbose: true }).some(move => move.to === square)) element.classList.add(piece ? "capture" : "legal");
    if (piece) { const span = document.createElement("span"); span.className = `piece ${piece.color === "w" ? "white-piece" : "black-piece"}`; span.textContent = icons[piece.type]; element.append(span); }
    element.addEventListener("click", () => clickSquare(square));
    board.append(element);
  }));
  renderMoves();
  if (chess.isCheckmate()) { status.textContent = "Checkmate"; hint.textContent = chess.turn() === "w" ? "Jev wins." : "You win!"; }
  else if (chess.isDraw()) { status.textContent = "Draw"; hint.textContent = "The game is over."; }
  else if (!thinking) { status.textContent = chess.turn() === "w" ? "Your move" : "Jev is thinking"; hint.textContent = chess.turn() === "w" ? "Select a piece to see legal moves." : "The bot is choosing from legal moves."; }
}

function renderMoves() {
  const history = chess.history();
  movesElement.replaceChildren();
  history.forEach((move, index) => { const el = document.createElement("div"); el.className = "move"; el.innerHTML = `<span class="move-no">${index % 2 === 0 ? `${Math.floor(index / 2) + 1}.` : ""}</span><span>${move}</span>`; movesElement.append(el); });
  moveCount.textContent = `${history.length} ${history.length === 1 ? "move" : "moves"}`;
}

function clickSquare(square) {
  if (thinking || chess.turn() !== "w" || chess.isGameOver()) return;
  const piece = chess.get(square);
  if (!selected && piece?.color === "w") { selected = square; render(); return; }
  if (selected) {
    try { chess.move({ from: selected, to: square, promotion: "q" }); selected = null; render(); requestBotMove(); }
    catch { selected = piece?.color === "w" ? square : null; render(); }
  }
}

async function requestBotMove() {
  if (chess.isGameOver() || chess.turn() !== "b") return;
  thinking = true; render();
  const legalMoves = Object.fromEntries(chess.moves({ verbose: true }).map(move => [move.from + move.to + (move.promotion || ""), `${move.from} to ${move.to}${move.promotion ? ` promote to ${names[move.promotion]}` : ""}`]));
  try {
    const response = await fetch("/api/bot-move", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ state: `Side to move: BLACK\nBoard:\n${boardState()}\nChoose one legal move.`, moves: legalMoves }) });
    if (!response.ok) throw new Error(await response.text());
    const { move } = await response.json();
    chess.move({ from: move.slice(0, 2), to: move.slice(2, 4), promotion: move.slice(4) || undefined });
  } catch (error) { status.textContent = "Bot unavailable"; hint.textContent = error.message || "Could not get a move."; }
  thinking = false; render();
}

document.querySelector("#new-game").addEventListener("click", () => { chess.reset(); selected = null; thinking = false; render(); });
render();
