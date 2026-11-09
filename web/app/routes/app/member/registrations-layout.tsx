import { Outlet } from "react-router";

export const handle = { crumb: "Mes inscriptions" };

export default function MyRegistrationsLayout() {
  return <Outlet />;
}
