import { useEffect, useState } from "react";
import type { FieldErrors, FieldValues, Path, UseFormRegister } from "react-hook-form";
import { Alert } from "~/ui/Alert";
import type { MemberFormValues } from "./form";

interface DuplicateConfirmProps<T extends MemberFormValues & FieldValues> {
  register: UseFormRegister<T>;
  errors: FieldErrors<T>;
}

// Doublon probable signalé par le serveur (research.md 005 › R8) : message et
// case « Enregistrer quand même », qui reçoit le focus à son apparition. Le
// message est conservé une fois affiché : react-hook-form efface l'erreur du
// champ dès qu'on coche la case, ce qui ferait disparaître celle-ci.
export function DuplicateConfirm<T extends MemberFormValues & FieldValues>({
  register,
  errors,
}: DuplicateConfirmProps<T>) {
  const error = errors.confirmDuplicate?.message as string | undefined;
  const [message, setMessage] = useState("");
  useEffect(() => {
    if (!error) return;
    setMessage(error);
    // Après le rendu de la case.
    requestAnimationFrame(() => document.getElementById("confirmer-doublon")?.focus());
  }, [error]);

  if (!message) return null;
  return (
    <Alert kind="warning" role="alert">
      <p id="doublon-message">{message}</p>
      <label className="mt-2 flex items-center gap-2 font-bold">
        <input
          type="checkbox"
          className="h-5 w-5"
          id="confirmer-doublon"
          aria-describedby="doublon-message"
          {...register("confirmDuplicate" as Path<T>)}
        />
        Enregistrer quand même
      </label>
    </Alert>
  );
}
