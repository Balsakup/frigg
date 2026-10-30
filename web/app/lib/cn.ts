import { type ClassValue, clsx } from "clsx";
import { twMerge } from "tailwind-merge";

// Compose des classes Tailwind : clsx gère les conditions, tailwind-merge
// résout les conflits (la dernière classe d'un même groupe l'emporte).
export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}
