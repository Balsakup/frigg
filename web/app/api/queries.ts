import createQueryHooks from "openapi-react-query";
import { client } from "./client";

// Hooks typés générés depuis le schéma OpenAPI.
export const $api = createQueryHooks(client);

// Session courante : chargée par le layout protégé, lue partout dans le shell.
export const sessionQuery = $api.queryOptions("get", "/session");
