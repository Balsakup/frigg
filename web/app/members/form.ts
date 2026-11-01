import type { components } from "~/api/schema";
import { formatPhone } from "~/lib/format";

export type Member = components["schemas"]["Member"];
export type AdminMember = components["schemas"]["AdminMember"];
type MemberBody = components["schemas"]["MyMemberBody"];

// Valeurs du formulaire d'une fiche : noms identiques aux clés JSON du corps,
// pour que les erreurs `body.<champ>` se placent sur leur champ.
export interface MemberFormValues {
  firstName: string;
  lastName: string;
  birthDate: string;
  email: string;
  phone: string;
  addressLine1: string;
  addressLine2: string;
  postalCode: string;
  city: string;
  confirmDuplicate: boolean;
}

export const memberFieldNames = [
  "firstName",
  "lastName",
  "birthDate",
  "email",
  "phone",
  "addressLine1",
  "addressLine2",
  "postalCode",
  "city",
  "confirmDuplicate",
] as const;

export const emptyMember: MemberFormValues = {
  firstName: "",
  lastName: "",
  birthDate: "",
  email: "",
  phone: "",
  addressLine1: "",
  addressLine2: "",
  postalCode: "",
  city: "",
  confirmDuplicate: false,
};

export function memberToForm(m: Member): MemberFormValues {
  return {
    firstName: m.firstName,
    lastName: m.lastName,
    birthDate: m.birthDate,
    email: m.email ?? "",
    phone: m.phone ? formatPhone(m.phone) : "",
    addressLine1: m.address?.line1 ?? "",
    addressLine2: m.address?.line2 ?? "",
    postalCode: m.address?.postalCode ?? "",
    city: m.address?.city ?? "",
    confirmDuplicate: false,
  };
}

// Corps commun envoyé à l'API ; le serveur nettoie et valide (principe VI).
export function formToBody(v: MemberFormValues): Omit<MemberBody, "isSelf"> {
  return {
    firstName: v.firstName,
    lastName: v.lastName,
    birthDate: v.birthDate,
    email: v.email,
    phone: v.phone,
    addressLine1: v.addressLine1,
    addressLine2: v.addressLine2,
    postalCode: v.postalCode,
    city: v.city,
    confirmDuplicate: v.confirmDuplicate,
  };
}

export function fullName(m: { firstName: string; lastName: string }) {
  return `${m.firstName} ${m.lastName}`;
}
