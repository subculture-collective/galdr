import { useCallback, useEffect, useState, type ReactNode } from "react";
import { Edit3, Save, Trash2, X } from "lucide-react";
import api from "@/lib/api";
import { relativeTime } from "@/lib/format";
import { useToast } from "@/contexts/ToastContext";

interface CustomerNoteAuthor {
  id: string;
  name: string;
  email: string;
  avatar_url: string;
}

interface CustomerNote {
  id: string;
  customer_id: string;
  user_id: string;
  author: CustomerNoteAuthor;
  content: string;
  can_edit: boolean;
  can_delete: boolean;
  created_at: string;
  updated_at: string;
}

interface CustomerNotesResponse {
  notes: CustomerNote[];
}

interface CustomerNotesProps {
  customerId: string;
}

export default function CustomerNotes({ customerId }: CustomerNotesProps) {
  const [notes, setNotes] = useState<CustomerNote[]>([]);
  const [loading, setLoading] = useState(true);
  const [content, setContent] = useState("");
  const [saving, setSaving] = useState(false);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [editingContent, setEditingContent] = useState("");
  const toast = useToast();

  const loadNotes = useCallback(async () => {
    try {
      const { data } = await api.get<CustomerNotesResponse>(
        `/customers/${customerId}/notes`,
      );
      setNotes(data.notes);
    } catch {
      toast.error("Failed to load notes");
    } finally {
      setLoading(false);
    }
  }, [customerId, toast]);

  useEffect(() => {
    loadNotes();
  }, [loadNotes]);

  async function addNote() {
    const trimmed = content.trim();
    if (!trimmed) return;
    setSaving(true);
    try {
      const { data } = await api.post<CustomerNote>(
        `/customers/${customerId}/notes`,
        { content: trimmed },
      );
      setNotes((current) => [data, ...current]);
      setContent("");
      toast.success("Note added");
    } catch {
      toast.error("Failed to add note");
    } finally {
      setSaving(false);
    }
  }

  async function saveEdit(noteId: string) {
    const trimmed = editingContent.trim();
    if (!trimmed) return;
    setSaving(true);
    try {
      const { data } = await api.put<CustomerNote>(
        `/customers/${customerId}/notes/${noteId}`,
        { content: trimmed },
      );
      setNotes((current) =>
        current.map((note) => (note.id === noteId ? data : note)),
      );
      setEditingId(null);
      setEditingContent("");
      toast.success("Note updated");
    } catch {
      toast.error("Failed to update note");
    } finally {
      setSaving(false);
    }
  }

  async function deleteNote(noteId: string) {
    setSaving(true);
    try {
      await api.delete(`/customers/${customerId}/notes/${noteId}`);
      setNotes((current) => current.filter((note) => note.id !== noteId));
      toast.success("Note deleted");
    } catch {
      toast.error("Failed to delete note");
    } finally {
      setSaving(false);
    }
  }

  function startEditing(note: CustomerNote) {
    setEditingId(note.id);
    setEditingContent(note.content);
  }

  function cancelEditing() {
    setEditingId(null);
  }

  function renderNotesContent() {
    if (loading) {
      return (
        <div className="galdr-card p-6 text-sm text-[var(--galdr-fg-muted)]">
          Loading notes...
        </div>
      );
    }

    if (notes.length === 0) {
      return (
        <div className="galdr-card p-6 text-center text-sm text-[var(--galdr-fg-muted)]">
          No notes yet. Add the first account note above.
        </div>
      );
    }

    return (
      <div className="space-y-3">
        {notes.map((note) => (
          <article key={note.id} className="galdr-card p-5">
            <div className="flex items-start justify-between gap-4">
              <div className="flex items-center gap-3">
                <div className="flex h-9 w-9 items-center justify-center rounded-full bg-[var(--galdr-surface-soft)] text-sm font-semibold text-[var(--galdr-fg)]">
                  {note.author.avatar_url ? (
                    <img
                      src={note.author.avatar_url}
                      alt=""
                      className="h-9 w-9 rounded-full object-cover"
                    />
                  ) : (
                    initials(note.author.name || note.author.email)
                  )}
                </div>
                <div>
                  <p className="text-sm font-medium text-[var(--galdr-fg)]">
                    {note.author.name || note.author.email}
                  </p>
                  <p className="text-xs text-[var(--galdr-fg-muted)]">
                    {relativeTime(note.created_at)}
                    {note.updated_at !== note.created_at ? " · edited" : ""}
                  </p>
                </div>
              </div>

              <div className="flex gap-2">
                {note.can_edit && editingId !== note.id && (
                  <button
                    onClick={() => startEditing(note)}
                    className="galdr-button-secondary px-3 py-2"
                    aria-label="Edit note"
                  >
                    <Edit3 className="h-4 w-4" />
                  </button>
                )}
                {note.can_delete && editingId !== note.id && (
                  <button
                    onClick={() => deleteNote(note.id)}
                    disabled={saving}
                    className="galdr-button-secondary px-3 py-2 text-[var(--galdr-danger)]"
                    aria-label="Delete note"
                  >
                    <Trash2 className="h-4 w-4" />
                  </button>
                )}
              </div>
            </div>

            {editingId === note.id ? (
              <div className="mt-4 space-y-3">
                <textarea
                  value={editingContent}
                  onChange={(event) => setEditingContent(event.target.value)}
                  rows={4}
                  className="w-full rounded-xl border border-[var(--galdr-border)] bg-[var(--galdr-surface)] px-4 py-3 text-sm text-[var(--galdr-fg)] outline-none transition focus:border-[var(--galdr-accent)]"
                />
                <div className="flex justify-end gap-2">
                  <button
                    onClick={cancelEditing}
                    className="galdr-button-secondary"
                  >
                    <X className="h-4 w-4" /> Cancel
                  </button>
                  <button
                    onClick={() => saveEdit(note.id)}
                    disabled={saving || !editingContent.trim()}
                    className="galdr-button-primary disabled:cursor-not-allowed disabled:opacity-50"
                  >
                    <Save className="h-4 w-4" /> Save
                  </button>
                </div>
              </div>
            ) : (
              <div className="mt-4 text-sm leading-6 text-[var(--galdr-fg)]">
                {renderMarkdown(note.content)}
              </div>
            )}
          </article>
        ))}
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <div className="galdr-card p-5">
        <label className="text-sm font-medium text-[var(--galdr-fg)]">
          Add a note
        </label>
        <textarea
          value={content}
          onChange={(event) => setContent(event.target.value)}
          rows={4}
          placeholder="Capture account context. Markdown supported: **bold**, *italic*, `code`, lists, links."
          className="mt-3 w-full rounded-xl border border-[var(--galdr-border)] bg-[var(--galdr-surface)] px-4 py-3 text-sm text-[var(--galdr-fg)] outline-none transition focus:border-[var(--galdr-accent)]"
        />
        <div className="mt-3 flex justify-end">
          <button
            onClick={addNote}
            disabled={saving || !content.trim()}
            className="galdr-button-primary disabled:cursor-not-allowed disabled:opacity-50"
          >
            Add note
          </button>
        </div>
      </div>

      {renderNotesContent()}
    </div>
  );
}

function initials(name: string): string {
  return name
    .split(" ")
    .filter(Boolean)
    .slice(0, 2)
    .map((part) => part[0]?.toUpperCase())
    .join("");
}

function renderMarkdown(content: string): ReactNode[] {
  const lines = content.split("\n");
  const nodes: ReactNode[] = [];
  let listItems: string[] = [];

  function flushList() {
    if (listItems.length === 0) return;
    const items = listItems;
    listItems = [];
    nodes.push(
      <ul key={`ul-${nodes.length}`} className="my-3 list-disc space-y-1 pl-5">
        {items.map((item, index) => (
          <li key={`${item}-${index}`}>{renderInlineMarkdown(item)}</li>
        ))}
      </ul>,
    );
  }

  lines.forEach((line, index) => {
    const trimmed = line.trim();
    if (trimmed.startsWith("- ")) {
      listItems.push(trimmed.slice(2));
      return;
    }

    flushList();
    if (!trimmed) {
      nodes.push(<div key={`space-${index}`} className="h-2" />);
      return;
    }
    if (trimmed.startsWith("### ")) {
      nodes.push(
        <h4 key={index} className="mt-3 font-semibold">
          {renderInlineMarkdown(trimmed.slice(4))}
        </h4>,
      );
      return;
    }
    if (trimmed.startsWith("## ")) {
      nodes.push(
        <h3 key={index} className="mt-3 text-base font-semibold">
          {renderInlineMarkdown(trimmed.slice(3))}
        </h3>,
      );
      return;
    }
    nodes.push(<p key={index}>{renderInlineMarkdown(trimmed)}</p>);
  });
  flushList();

  return nodes;
}

function renderInlineMarkdown(text: string): ReactNode[] {
  const pattern = /(\*\*[^*]+\*\*|\*[^*]+\*|`[^`]+`|\[[^\]]+\]\([^\s)]+\))/g;
  return text.split(pattern).map((part, index) => {
    if (part.startsWith("**") && part.endsWith("**")) {
      return <strong key={index}>{part.slice(2, -2)}</strong>;
    }
    if (part.startsWith("*") && part.endsWith("*")) {
      return <em key={index}>{part.slice(1, -1)}</em>;
    }
    if (part.startsWith("`") && part.endsWith("`")) {
      return (
        <code
          key={index}
          className="rounded bg-[var(--galdr-surface-soft)] px-1.5 py-0.5 text-xs"
        >
          {part.slice(1, -1)}
        </code>
      );
    }
    if (part.startsWith("[")) {
      const match = part.match(/^\[([^\]]+)\]\(([^\s)]+)\)$/);
      const href = match?.[2] ?? "";
      if (match && /^(https?:|mailto:)/.test(href)) {
        return (
          <a
            key={index}
            href={href}
            className="galdr-link"
            rel="noreferrer"
            target={href.startsWith("mailto:") ? undefined : "_blank"}
          >
            {match[1]}
          </a>
        );
      }
    }
    return part;
  });
}
