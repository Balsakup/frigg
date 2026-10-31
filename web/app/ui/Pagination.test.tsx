import { render, screen } from "@testing-library/react";
import { createMemoryRouter, RouterProvider } from "react-router";
import { describe, expect, it } from "vitest";
import { Pagination } from "./Pagination";

function renderPagination(page: number, pageCount: number) {
  const router = createMemoryRouter([
    {
      path: "/",
      element: <Pagination page={page} pageCount={pageCount} hrefFor={(p) => `/?page=${p}`} />,
    },
  ]);
  return render(<RouterProvider router={router} />);
}

describe("Pagination", () => {
  it("affiche la page courante et les liens voisins", () => {
    renderPagination(2, 5);
    const nav = screen.getByRole("navigation", { name: "Pagination" });
    expect(nav.textContent).toContain("Page 2 sur 5");
    expect(screen.getByText("Page 2 sur 5").getAttribute("aria-current")).toBe("page");
    expect(screen.getByRole("link", { name: /Précédente/ }).getAttribute("href")).toBe("/?page=1");
    expect(screen.getByRole("link", { name: /Suivante/ }).getAttribute("href")).toBe("/?page=3");
  });

  it("n'affiche pas de lien vers une page inexistante", () => {
    renderPagination(1, 3);
    expect(screen.queryByRole("link", { name: /Précédente/ })).toBeNull();
    expect(screen.getByRole("link", { name: /Suivante/ })).toBeTruthy();
  });

  it("ne s'affiche pas s'il n'y a qu'une page", () => {
    const { container } = renderPagination(1, 1);
    expect(container.innerHTML).toBe("");
  });
});
