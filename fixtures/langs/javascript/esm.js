const MAX = 5;
export const LIMIT = 10;
export let counter = 0;
export var legacy = 1;
export const settings = { a: 1 };
export const Banner = memo(() => 1);
const Ctx = createContext(null);
const Lazy = React.lazy(() => import('./x'));
export const run = () => 1;
const Plain = class {};
const computed = compute();

export function load() {
  const inner = () => 2;
  const value = 3;
  return inner() + value;
}

export class Store {
  get(key) { return key; }
}
