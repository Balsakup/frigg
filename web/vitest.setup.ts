import { cleanup } from "@testing-library/react";
import { afterEach } from "vitest";

// Sans `globals: true`, Testing Library ne démonte pas les composants seul.
afterEach(() => {
  cleanup();
});
