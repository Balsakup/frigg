// Mode SPA (ssr: false) : ce point d'entrée ne sert qu'au build, pour pré-rendre
// index.html. En production, c'est le binaire Go qui sert les fichiers statiques.
// Le fournir explicitement évite que React Router réinjecte `isbot` dans package.json.

import { renderToReadableStream } from "react-dom/server";
import type { EntryContext } from "react-router";
import { ServerRouter } from "react-router";

export default async function handleRequest(
  request: Request,
  responseStatusCode: number,
  responseHeaders: Headers,
  routerContext: EntryContext,
) {
  const body = await renderToReadableStream(
    <ServerRouter context={routerContext} url={request.url} />,
    { signal: request.signal },
  );
  await body.allReady;

  responseHeaders.set("Content-Type", "text/html");
  return new Response(body, {
    headers: responseHeaders,
    status: responseStatusCode,
  });
}
