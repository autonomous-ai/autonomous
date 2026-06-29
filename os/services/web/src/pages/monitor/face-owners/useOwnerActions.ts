import { useEffect, useRef, useState } from "react";
import { HW } from "../types";

// Owner-mutation flows for the Users page: enroll a new user, rename, and remove
// a user / photo / voice sample (each with its themed confirm dialog). Takes the
// owners-list `refresh` so it can reload after a successful change.
export function useOwnerActions(refresh: () => void) {
  // Enroll form state
  const [showEnroll, setShowEnroll] = useState(false);
  const [enrollName, setEnrollName] = useState("");
  const [enrollTgUsername, setEnrollTgUsername] = useState("");
  const [enrollTgId, setEnrollTgId] = useState("");
  const [enrollFile, setEnrollFile] = useState<File | null>(null);
  // Object-URL preview of the chosen photo so the operator sees the actual face
  // before enrolling. Recreated on file change, revoked on cleanup to avoid leaks.
  const [enrollPreview, setEnrollPreview] = useState<string | null>(null);
  const [enrolling, setEnrolling] = useState(false);
  const [enrollError, setEnrollError] = useState("");
  const [enrollDragging, setEnrollDragging] = useState(false);

  const fileInputRef = useRef<HTMLInputElement | null>(null);

  // Rename modal state. `renaming` holds the current label being edited (null =
  // closed); replaces the native prompt()/alert() with a themed dialog.
  const [renaming, setRenaming] = useState<string | null>(null);
  const [renameValue, setRenameValue] = useState("");
  const [renameError, setRenameError] = useState("");
  const [renameSaving, setRenameSaving] = useState(false);

  // Label pending the themed delete confirmation (null = no dialog open). Used
  // instead of window.confirm() so the prompt matches the dark-amber theme.
  const [confirmDelete, setConfirmDelete] = useState<string | null>(null);
  // Photo pending themed delete confirmation: { label, filename } or null.
  const [confirmPhoto, setConfirmPhoto] = useState<{ label: string; filename: string } | null>(null);
  // Voice sample pending themed delete confirmation: { label, filename } or null.
  const [confirmVoice, setConfirmVoice] = useState<{ label: string; filename: string } | null>(null);

  // Delete-in-flight state
  const [deleting, setDeleting] = useState<string | null>(null);
  const [deletingPhoto, setDeletingPhoto] = useState<string | null>(null); // "label/filename"

  useEffect(() => {
    if (!enrollFile) { setEnrollPreview(null); return; }
    const url = URL.createObjectURL(enrollFile);
    setEnrollPreview(url);
    return () => URL.revokeObjectURL(url);
  }, [enrollFile]);

  // Close the enroll / rename modals on Escape. (Confirm dialogs handle their
  // own Escape inside ConfirmDialog.)
  useEffect(() => {
    if (!showEnroll && renaming == null) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") { setShowEnroll(false); setRenaming(null); }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [showEnroll, renaming]);

  const handleEnroll = async () => {
    if (!enrollFile || !enrollName.trim()) return;
    setEnrolling(true);
    setEnrollError("");
    try {
      const base64 = await new Promise<string>((resolve, reject) => {
        const reader = new FileReader();
        reader.onload = () => {
          const result = reader.result as string;
          resolve(result.split(",")[1]); // strip "data:image/...;base64,"
        };
        reader.onerror = () => reject(new Error("Failed to read file"));
        reader.readAsDataURL(enrollFile);
      });
      const res = await fetch(`${HW}/face/enroll`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          image_base64: base64,
          label: enrollName.trim().toLowerCase(),
          ...(enrollTgUsername.trim() ? { telegram_username: enrollTgUsername.trim() } : {}),
          ...(enrollTgId.trim() ? { telegram_id: enrollTgId.trim() } : {}),
        }),
      });
      if (!res.ok) {
        const err = await res.json().catch(() => ({ detail: "Unknown error" }));
        throw new Error(err.detail || `HTTP ${res.status}`);
      }
      setShowEnroll(false);
      setEnrollName("");
      setEnrollTgUsername("");
      setEnrollTgId("");
      setEnrollFile(null);
      if (fileInputRef.current) fileInputRef.current.value = "";
      refresh();
    } catch (e) {
      setEnrollError((e as Error).message);
    } finally {
      setEnrolling(false);
    }
  };

  // Open the themed confirm dialog for removing a user.
  const handleRemove = (label: string) => setConfirmDelete(label);

  // Execute the removal once confirmed in the dialog.
  const confirmRemove = async () => {
    const label = confirmDelete;
    if (label == null) return;
    setConfirmDelete(null);
    setDeleting(label);
    try {
      await fetch(`${HW}/face/remove`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ label }),
      });
      refresh();
    } catch {
      // ignore
    } finally {
      setDeleting(null);
    }
  };

  // Open the themed rename dialog, seeded with the current label.
  const handleRename = (oldLabel: string) => {
    setRenaming(oldLabel);
    setRenameValue(oldLabel);
    setRenameError("");
  };

  // Commit the rename from the dialog. Validation surfaces inline (no alert),
  // and the dialog closes on success.
  const submitRename = async () => {
    if (renaming == null) return;
    const oldLabel = renaming;
    const newLabel = renameValue.trim().toLowerCase();
    if (!newLabel || newLabel === oldLabel) { setRenaming(null); return; }
    if (!/^[a-z0-9_-]+$/.test(newLabel)) {
      setRenameError("Use only lowercase letters, digits, _ and -");
      return;
    }
    setRenameSaving(true);
    setRenameError("");
    try {
      const resp = await fetch(`${HW}/users/rename`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ old_label: oldLabel, new_label: newLabel }),
      });
      if (!resp.ok) {
        const data = await resp.json().catch(() => ({}));
        setRenameError(`Rename failed: ${data.detail ?? resp.statusText}`);
        setRenameSaving(false);
        return;
      }
      setRenaming(null);
      refresh();
    } catch (e) {
      setRenameError(`Rename failed: ${e instanceof Error ? e.message : String(e)}`);
    }
    setRenameSaving(false);
  };

  // Open the themed confirm dialog for removing a single photo.
  const handleRemovePhoto = (label: string, filename: string) => setConfirmPhoto({ label, filename });

  // Execute the photo removal once confirmed in the dialog.
  const confirmRemovePhoto = async () => {
    if (!confirmPhoto) return;
    const { label, filename } = confirmPhoto;
    setConfirmPhoto(null);
    const key = `${label}/${filename}`;
    setDeletingPhoto(key);
    try {
      await fetch(`${HW}/face/photo/remove`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ label, filename }),
      });
      refresh();
    } catch {
      // ignore
    } finally {
      setDeletingPhoto(null);
    }
  };

  // Voice sample delete — only audio files. JSON/NPY (metadata, embedding
  // cache) are protected because deleting them silently corrupts the
  // speaker_recognizer profile. Backend OS server /api/voice/file/remove
  // re-enrolls from remaining samples to refresh the embedding.
  // Open the themed confirm dialog for removing a voice sample.
  const handleRemoveVoiceFile = (label: string, filename: string) => setConfirmVoice({ label, filename });

  // Execute the voice-sample removal once confirmed.
  const confirmRemoveVoice = async () => {
    if (!confirmVoice) return;
    const { label, filename } = confirmVoice;
    setConfirmVoice(null);
    const key = `${label}/voice/${filename}`;
    setDeletingPhoto(key);
    try {
      await fetch(`/api/voice/file/remove`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ name: label, file: filename }),
      });
      refresh();
    } catch {
      // ignore
    } finally {
      setDeletingPhoto(null);
    }
  };

  return {
    // enroll
    showEnroll, setShowEnroll,
    enrollName, setEnrollName,
    enrollTgUsername, setEnrollTgUsername,
    enrollTgId, setEnrollTgId,
    enrollFile, setEnrollFile,
    enrollPreview, enrolling, enrollError,
    enrollDragging, setEnrollDragging,
    fileInputRef,
    handleEnroll,
    // rename
    renaming, setRenaming,
    renameValue, setRenameValue,
    renameError, setRenameError,
    renameSaving,
    handleRename, submitRename,
    // delete user / photo / voice
    confirmDelete, setConfirmDelete,
    confirmPhoto, setConfirmPhoto,
    confirmVoice, setConfirmVoice,
    deleting, deletingPhoto,
    handleRemove, confirmRemove,
    handleRemovePhoto, confirmRemovePhoto,
    handleRemoveVoiceFile, confirmRemoveVoice,
  };
}
