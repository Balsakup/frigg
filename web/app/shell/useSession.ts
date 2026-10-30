import { useQuery } from "@tanstack/react-query";
import { sessionQuery } from "~/api/queries";

// Session courante ; toujours présente sous le layout protégé, qui la précharge.
export function useSession() {
  const { data } = useQuery(sessionQuery);
  if (!data) {
    throw new Error("useSession doit être utilisé sous le layout protégé");
  }
  return data;
}
