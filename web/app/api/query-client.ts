import { QueryClient } from "@tanstack/react-query";

// Instance unique, partagée par les loaders (ensureQueryData) et les composants.
export const queryClient = new QueryClient({
  defaultOptions: {
    queries: { retry: false, staleTime: 30_000, refetchOnWindowFocus: true },
  },
});
