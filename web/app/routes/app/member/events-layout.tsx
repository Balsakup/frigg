import { Outlet } from "react-router";

export const handle = { crumb: "Événements à venir" };

export default function MyEventsLayout() {
  return <Outlet />;
}
