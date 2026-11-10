// « 1 ligne », « 3 lignes ».
export function plural(n: number, one: string, many = `${one}s`) {
  return `${n.toLocaleString("fr-FR")} ${n > 1 ? many : one}`;
}

export const separatorLabels: Record<string, string> = {
  ";": "point-virgule",
  ",": "virgule",
  "\t": "tabulation",
};

export const encodingLabels: Record<string, string> = {
  "utf-8": "UTF-8",
  "windows-1252": "Windows-1252",
};
