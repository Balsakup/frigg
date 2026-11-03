// Lecture des e-mails capturés par Mailpit (http://localhost:8025).
const MAILPIT = "http://localhost:8025/api/v1";

interface Summary {
  ID: string;
  Created: string;
}

// Attend le prochain e-mail adressé à `to` reçu après `since`, et renvoie le
// lien qu'il contient.
export async function waitForLink(to: string, since: Date, timeoutMs = 15_000): Promise<string> {
  const deadline = Date.now() + timeoutMs;
  while (Date.now() < deadline) {
    const res = await fetch(`${MAILPIT}/search?query=${encodeURIComponent(`to:"${to}"`)}`);
    if (res.ok) {
      const { messages } = (await res.json()) as { messages: Summary[] };
      const latest = messages.find((m) => new Date(m.Created) >= since);
      if (latest) {
        const msg = (await (await fetch(`${MAILPIT}/message/${latest.ID}`)).json()) as {
          Text: string;
        };
        const link = msg.Text.match(/https?:\/\/\S+#[A-Za-z0-9_-]+/)?.[0];
        if (link) return link;
      }
    }
    await new Promise((r) => setTimeout(r, 300));
  }
  throw new Error(`aucun e-mail reçu pour ${to}`);
}

// Attend le prochain e-mail adressé à `to` reçu après `since` dont l'objet
// contient `subject`, et renvoie son texte (e-mails sans jeton, 006).
export async function waitForMail(
  to: string,
  since: Date,
  subject: string,
  timeoutMs = 15_000,
): Promise<string> {
  const deadline = Date.now() + timeoutMs;
  while (Date.now() < deadline) {
    const res = await fetch(`${MAILPIT}/search?query=${encodeURIComponent(`to:"${to}"`)}`);
    if (res.ok) {
      const { messages } = (await res.json()) as { messages: (Summary & { Subject: string })[] };
      const found = messages.find(
        (m) => new Date(m.Created) >= since && m.Subject.includes(subject),
      );
      if (found) {
        const msg = (await (await fetch(`${MAILPIT}/message/${found.ID}`)).json()) as {
          Text: string;
        };
        return msg.Text;
      }
    }
    await new Promise((r) => setTimeout(r, 300));
  }
  throw new Error(`aucun e-mail « ${subject} » reçu pour ${to}`);
}

export interface MailpitMessage {
  From: { Name: string; Address: string };
  To: { Name: string; Address: string }[];
  Subject: string;
  Text: string;
  Attachments: { FileName: string; ContentType: string; Size: number }[];
}

// Attend les e-mails dont l'objet contient `subject` reçus après `since`, et
// renvoie le détail de chacun (communications, 009).
export async function waitForMessages(
  subject: string,
  since: Date,
  count: number,
  timeoutMs = 20_000,
): Promise<MailpitMessage[]> {
  const deadline = Date.now() + timeoutMs;
  let found: (Summary & { Subject: string })[] = [];
  while (Date.now() < deadline) {
    const res = await fetch(
      `${MAILPIT}/search?query=${encodeURIComponent(`subject:"${subject}"`)}`,
    );
    if (res.ok) {
      const { messages } = (await res.json()) as { messages: (Summary & { Subject: string })[] };
      found = messages.filter((m) => new Date(m.Created) >= since);
      if (found.length >= count) break;
    }
    await new Promise((r) => setTimeout(r, 300));
  }
  return Promise.all(
    found.map(
      async (m) => (await (await fetch(`${MAILPIT}/message/${m.ID}`)).json()) as MailpitMessage,
    ),
  );
}

export interface MailpitFullMessage extends MailpitMessage {
  HTML: string;
  Inline: { FileName: string; ContentType: string; ContentID: string }[];
}

// Attend le prochain e-mail adressé à `to` reçu après `since` dont l'objet
// contient `subject`, et renvoie son détail complet, HTML et images
// embarquées comprises (identité visuelle, 010).
export async function waitForFullMail(
  to: string,
  since: Date,
  subject: string,
  timeoutMs = 15_000,
): Promise<MailpitFullMessage> {
  const deadline = Date.now() + timeoutMs;
  while (Date.now() < deadline) {
    const res = await fetch(`${MAILPIT}/search?query=${encodeURIComponent(`to:"${to}"`)}`);
    if (res.ok) {
      const { messages } = (await res.json()) as { messages: (Summary & { Subject: string })[] };
      const found = messages.find(
        (m) => new Date(m.Created) >= since && m.Subject.includes(subject),
      );
      if (found) {
        return (await (await fetch(`${MAILPIT}/message/${found.ID}`)).json()) as MailpitFullMessage;
      }
    }
    await new Promise((r) => setTimeout(r, 300));
  }
  throw new Error(`aucun e-mail « ${subject} » reçu pour ${to}`);
}
