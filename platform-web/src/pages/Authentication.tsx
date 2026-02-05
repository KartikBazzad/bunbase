import { useState, useEffect, useMemo, useCallback, useRef } from "react";
import { useParams } from "react-router-dom";
import {
  Search,
  Download,
  UserPlus,
  MoreVertical,
  Mail,
  Code2,
  Phone,
  UserX,
  Info,
} from "lucide-react";
import { api } from "../lib/api";

type TabId = "users" | "signin";

interface AuthUser {
  id: string;
  user_id?: string;
  project_id: string;
  email: string;
  created_at?: string;
}

const PROVIDER_META: Record<
  string,
  { label: string; description: string; icon: typeof Mail }
> = {
  email_password: {
    label: "Email / Password",
    description: "Native email and password authentication.",
    icon: Mail,
  },
  google: {
    label: "Google",
    description: "OAuth2 login with Google accounts.",
    icon: Code2,
  },
  github: {
    label: "GitHub",
    description: "Authenticate developers with GitHub.",
    icon: Code2,
  },
  phone: {
    label: "Phone",
    description: "SMS-based one-time password login.",
    icon: Phone,
  },
  anonymous: {
    label: "Anonymous",
    description: "Let users try your app without signing in.",
    icon: UserX,
  },
};

const DEFAULT_PROVIDER_KEYS = [
  "email_password",
  "google",
  "github",
  "phone",
  "anonymous",
];

function formatDate(value: string | undefined): string {
  if (!value) return "—";
  try {
    const d = new Date(value);
    return d.toLocaleDateString(undefined, {
      year: "numeric",
      month: "short",
      day: "numeric",
    });
  } catch {
    return "—";
  }
}

function truncateUid(uid: string, maxLen = 12): string {
  if (!uid || uid.length <= maxLen) return uid;
  return uid.slice(0, 8) + "…";
}

export function Authentication() {
  const { id: projectId } = useParams<{ id: string }>();
  const [tab, setTab] = useState<TabId>("users");

  // Users tab state
  const [users, setUsers] = useState<AuthUser[]>([]);
  const [usersLoading, setUsersLoading] = useState(false);
  const [usersError, setUsersError] = useState("");
  const [searchQuery, setSearchQuery] = useState("");
  const [showAddUserModal, setShowAddUserModal] = useState(false);
  const [addUserEmail, setAddUserEmail] = useState("");
  const [addUserPassword, setAddUserPassword] = useState("");
  const [addUserLoading, setAddUserLoading] = useState(false);
  const [addUserError, setAddUserError] = useState("");

  // Sign-in methods state
  const [config, setConfig] = useState<{
    providers?: Record<string, { enabled?: boolean; [k: string]: unknown }>;
    rate_limit?: Record<string, unknown>;
  }>({});
  const [configLoading, setConfigLoading] = useState(false);
  const [configError, setConfigError] = useState("");
  const [configSaving, setConfigSaving] = useState(false);

  // Actions popover (single popover for all rows so it floats above table)
  const [actionsPopoverUser, setActionsPopoverUser] = useState<AuthUser | null>(
    null,
  );
  const [actionsPopoverAnchor, setActionsPopoverAnchor] =
    useState<DOMRect | null>(null);
  const actionsPopoverRef = useRef<HTMLDivElement>(null);

  const openActionsPopover = useCallback(
    (e: React.MouseEvent, user: AuthUser) => {
      e.stopPropagation();
      setActionsPopoverAnchor(
        (e.currentTarget as HTMLElement).getBoundingClientRect(),
      );
      setActionsPopoverUser(user);
    },
    [],
  );

  const closeActionsPopover = useCallback(() => {
    // Only hide; state is cleared in onToggle when popover actually closes (keeps menu content visible until then)
    actionsPopoverRef.current?.hidePopover();
  }, []);

  useEffect(() => {
    if (
      !actionsPopoverUser ||
      !actionsPopoverAnchor ||
      !actionsPopoverRef.current
    )
      return;
    const el = actionsPopoverRef.current;
    el.style.position = "fixed";
    el.style.left = `${actionsPopoverAnchor.right}px`;
    el.style.top = `${actionsPopoverAnchor.bottom + 4}px`;
    el.style.transform = "translateX(-100%)";
    el.showPopover();
  }, [actionsPopoverUser, actionsPopoverAnchor]);

  const refetchUsers = useCallback(() => {
    if (!projectId) return;
    setUsersLoading(true);
    api
      .listProjectAuthUsers(projectId)
      .then((res) => {
        setUsers(res.users || []);
        if (res.error) setUsersError(res.error);
      })
      .catch((err) =>
        setUsersError(
          err instanceof Error ? err.message : "Failed to load users",
        ),
      )
      .finally(() => setUsersLoading(false));
  }, [projectId]);

  useEffect(() => {
    if (!projectId || tab !== "users") return;
    setUsersError("");
    refetchUsers();
  }, [projectId, tab, refetchUsers]);

  useEffect(() => {
    if (!projectId || tab !== "signin") return;
    setConfigError("");
    setConfigLoading(true);
    api
      .getProjectAuthConfig(projectId)
      .then((c) => {
        setConfig({
          providers:
            c.providers && typeof c.providers === "object"
              ? (c.providers as Record<string, { enabled?: boolean }>)
              : {},
          rate_limit: c.rate_limit,
        });
        if (c.error) setConfigError(c.error);
      })
      .catch((err) =>
        setConfigError(
          err instanceof Error ? err.message : "Failed to load config",
        ),
      )
      .finally(() => setConfigLoading(false));
  }, [projectId, tab]);

  const filteredUsers = useMemo(() => {
    if (!searchQuery.trim()) return users;
    const q = searchQuery.trim().toLowerCase();
    return users.filter(
      (u) =>
        u.email?.toLowerCase().includes(q) ||
        u.id?.toLowerCase().includes(q) ||
        u.user_id?.toLowerCase().includes(q),
    );
  }, [users, searchQuery]);

  const toggleProvider = async (providerKey: string, enabled: boolean) => {
    if (!projectId) return;
    const next = {
      ...config,
      providers: {
        ...(config.providers || {}),
        [providerKey]: {
          ...((config.providers || {})[providerKey] as object),
          enabled,
        },
      },
    };
    setConfigSaving(true);
    try {
      await api.updateProjectAuthConfig(projectId, next);
      setConfig(next);
    } catch (err) {
      setConfigError(
        err instanceof Error ? err.message : "Failed to update config",
      );
    } finally {
      setConfigSaving(false);
    }
  };

  const pageSize = 10;
  const [usersPage, setUsersPage] = useState(0);
  const paginatedUsers = useMemo(
    () =>
      filteredUsers.slice(
        usersPage * pageSize,
        usersPage * pageSize + pageSize,
      ),
    [filteredUsers, usersPage],
  );
  const totalPages = Math.max(1, Math.ceil(filteredUsers.length / pageSize));
  const newUsers24h = useMemo(() => {
    const dayAgo = Date.now() - 24 * 60 * 60 * 1000;
    return users.filter((u) => {
      const t = u.created_at ? new Date(u.created_at).getTime() : 0;
      return t >= dayAgo;
    }).length;
  }, [users]);

  if (!projectId) {
    return (
      <div className="min-h-screen bg-base-200 flex flex-col">
        <main className="container mx-auto px-4 sm:px-6 lg:px-8 max-w-7xl py-8">
          <p className="text-base-content/70">No project selected.</p>
        </main>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-base-200 flex flex-col">
      <main className="container mx-auto px-4 sm:px-6 lg:px-8 max-w-7xl py-8">
        <div className="mb-6">
          <h1 className="text-2xl font-bold">Authentication</h1>
          <p className="text-base-content/70 mt-1">
            Manage your application users, providers and security policies.
          </p>
        </div>

        <div className="tabs tabs-boxed bg-base-100 p-1 rounded-lg inline-flex mb-6">
          <button
            type="button"
            className={`tab ${tab === "users" ? "tab-active" : ""}`}
            onClick={() => setTab("users")}
          >
            Users
          </button>
          <button
            type="button"
            className={`tab ${tab === "signin" ? "tab-active" : ""}`}
            onClick={() => setTab("signin")}
          >
            Sign-in Methods
          </button>
        </div>

        {tab === "users" && (
          <div className="space-y-6">
            <div className="flex flex-wrap items-center justify-between gap-4">
              <div className="flex flex-wrap items-center gap-3">
                <button
                  type="button"
                  className="btn btn-ghost btn-sm gap-2"
                  disabled
                >
                  <Download className="w-4 h-4" />
                  Export Users
                </button>
                <button
                  type="button"
                  className="btn btn-primary btn-sm gap-2"
                  onClick={() => {
                    setAddUserError("");
                    setAddUserEmail("");
                    setAddUserPassword("");
                    setShowAddUserModal(true);
                  }}
                >
                  <UserPlus className="w-4 h-4" />
                  Add User
                </button>
              </div>
            </div>

            <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
              <div className="card bg-base-100 shadow-md">
                <div className="card-body py-4">
                  <p className="text-sm text-base-content/60">Total Users</p>
                  <p className="text-2xl font-bold">{users.length}</p>
                </div>
              </div>
              <div className="card bg-base-100 shadow-md">
                <div className="card-body py-4">
                  <p className="text-sm text-base-content/60">
                    Active Sessions
                  </p>
                  <p className="text-2xl font-bold">—</p>
                  <span className="badge badge-ghost badge-sm">
                    Not yet tracked
                  </span>
                </div>
              </div>
              <div className="card bg-base-100 shadow-md">
                <div className="card-body py-4">
                  <p className="text-sm text-base-content/60">
                    New Users (24h)
                  </p>
                  <p className="text-2xl font-bold">+{newUsers24h}</p>
                  <span className="badge badge-ghost badge-sm">Stable</span>
                </div>
              </div>
            </div>

            <div className="card bg-base-100 shadow-md">
              <div className="card-body">
                <div className="flex flex-wrap items-center gap-4 mb-4">
                  <div className="relative flex-1 min-w-[200px]">
                    <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-base-content/50" />
                    <input
                      type="text"
                      placeholder="Search by email or UID…"
                      className="input input-bordered w-full pl-9"
                      value={searchQuery}
                      onChange={(e) => {
                        setSearchQuery(e.target.value);
                        setUsersPage(0);
                      }}
                    />
                  </div>
                </div>

                {usersError && (
                  <div className="alert alert-error text-sm mb-4">
                    {usersError}
                  </div>
                )}

                {usersLoading ? (
                  <div className="flex justify-center py-12">
                    <span className="loading loading-spinner loading-md" />
                  </div>
                ) : (
                  <>
                    <div className="overflow-x-auto">
                      <table className="table table-pin-rows">
                        <thead>
                          <tr>
                            <th>Identifier</th>
                            <th>Created</th>
                            <th>User UID</th>
                            <th className="w-20">Actions</th>
                          </tr>
                        </thead>
                        <tbody>
                          {paginatedUsers.length === 0 ? (
                            <tr>
                              <td
                                colSpan={4}
                                className="text-center text-base-content/60 py-8"
                              >
                                {usersError && users.length === 0
                                  ? "Could not load users. See message above."
                                  : filteredUsers.length === 0 &&
                                      users.length > 0
                                    ? "No users match your search."
                                    : "No users yet."}
                              </td>
                            </tr>
                          ) : (
                            paginatedUsers.map((u) => (
                              <tr key={u.id}>
                                <td>
                                  <span className="font-medium">
                                    {u.email || u.id}
                                  </span>
                                </td>
                                <td>{formatDate(u.created_at)}</td>
                                <td>
                                  <code className="text-xs bg-base-200 px-1.5 py-0.5 rounded">
                                    {truncateUid(u.user_id || u.id)}
                                  </code>
                                </td>
                                <td>
                                  <button
                                    type="button"
                                    className="btn btn-ghost btn-xs btn-square"
                                    onClick={(e) => openActionsPopover(e, u)}
                                    aria-haspopup="menu"
                                    aria-expanded={
                                      actionsPopoverUser?.id === u.id
                                    }
                                  >
                                    <MoreVertical className="w-4 h-4" />
                                  </button>
                                </td>
                              </tr>
                            ))
                          )}
                        </tbody>
                      </table>
                    </div>
                    {/* Actions menu: single popover in top layer so it isn't clipped by table overflow */}
                    <div
                      ref={actionsPopoverRef}
                      id="user-actions-popover"
                      popover="auto"
                      className="menu bg-base-200 rounded-box z-50 w-40 p-1 shadow-lg border border-base-300"
                      onToggle={(e) => {
                        if ((e as ToggleEvent).newState === "closed") {
                          setActionsPopoverUser(null);
                          setActionsPopoverAnchor(null);
                        }
                      }}
                    >
                      {actionsPopoverUser && (
                        <>
                          <li>
                            <button type="button" onClick={closeActionsPopover}>
                              View
                            </button>
                          </li>
                          <li>
                            <button type="button" onClick={closeActionsPopover}>
                              Disable
                            </button>
                          </li>
                        </>
                      )}
                    </div>
                    {totalPages > 1 && (
                      <div className="flex items-center justify-between mt-4 pt-4 border-t border-base-300">
                        <p className="text-sm text-base-content/60">
                          Showing {usersPage * pageSize + 1}–
                          {Math.min(
                            (usersPage + 1) * pageSize,
                            filteredUsers.length,
                          )}{" "}
                          of {filteredUsers.length} users
                        </p>
                        <div className="join">
                          <button
                            type="button"
                            className="join-item btn btn-sm"
                            disabled={usersPage === 0}
                            onClick={() =>
                              setUsersPage((p) => Math.max(0, p - 1))
                            }
                          >
                            Previous
                          </button>
                          <button
                            type="button"
                            className="join-item btn btn-sm"
                            disabled={usersPage >= totalPages - 1}
                            onClick={() =>
                              setUsersPage((p) =>
                                Math.min(totalPages - 1, p + 1),
                              )
                            }
                          >
                            Next
                          </button>
                        </div>
                      </div>
                    )}
                  </>
                )}
              </div>
            </div>
          </div>
        )}

        {/* Add User modal */}
        <dialog className={`modal ${showAddUserModal ? "modal-open" : ""}`}>
          <div className="modal-box">
            <h3 className="font-bold text-lg">Add user</h3>
            <p className="text-sm text-base-content/60 mt-1">
              Create a new user with email and password. Email/password sign-in
              must be enabled in Sign-in methods.
            </p>
            <form
              className="space-y-4 mt-4"
              onSubmit={async (e) => {
                e.preventDefault();
                const email = addUserEmail.trim();
                const password = addUserPassword;
                if (!email || !password) {
                  setAddUserError("Email and password are required.");
                  return;
                }
                if (password.length < 6) {
                  setAddUserError("Password must be at least 6 characters.");
                  return;
                }
                setAddUserError("");
                setAddUserLoading(true);
                try {
                  await api.createProjectAuthUser(projectId!, {
                    email,
                    password,
                  });
                  setShowAddUserModal(false);
                  setAddUserEmail("");
                  setAddUserPassword("");
                  refetchUsers();
                } catch (err) {
                  setAddUserError(
                    err instanceof Error
                      ? err.message
                      : "Failed to create user",
                  );
                } finally {
                  setAddUserLoading(false);
                }
              }}
            >
              <div className="form-control">
                <label className="label" htmlFor="add-user-email">
                  <span className="label-text">Email</span>
                </label>
                <input
                  id="add-user-email"
                  type="email"
                  placeholder="user@example.com"
                  className="input input-bordered w-full"
                  value={addUserEmail}
                  onChange={(e) => setAddUserEmail(e.target.value)}
                  autoComplete="email"
                  disabled={addUserLoading}
                />
              </div>
              <div className="form-control">
                <label className="label" htmlFor="add-user-password">
                  <span className="label-text">Password</span>
                </label>
                <input
                  id="add-user-password"
                  type="password"
                  placeholder="••••••••"
                  className="input input-bordered w-full"
                  value={addUserPassword}
                  onChange={(e) => setAddUserPassword(e.target.value)}
                  autoComplete="new-password"
                  disabled={addUserLoading}
                />
                <p className="text-xs text-base-content/50 mt-1">
                  Minimum 6 characters
                </p>
              </div>
              {addUserError && (
                <div className="alert alert-error text-sm">{addUserError}</div>
              )}
              <div className="modal-action">
                <button
                  type="button"
                  className="btn btn-ghost"
                  onClick={() => {
                    setShowAddUserModal(false);
                    setAddUserError("");
                  }}
                  disabled={addUserLoading}
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  className="btn btn-primary"
                  disabled={addUserLoading}
                >
                  {addUserLoading ? (
                    <>
                      <span className="loading loading-spinner loading-sm" />
                      Creating…
                    </>
                  ) : (
                    "Add user"
                  )}
                </button>
              </div>
            </form>
          </div>
          <form method="dialog" className="modal-backdrop">
            <button
              type="button"
              onClick={() => {
                setShowAddUserModal(false);
                setAddUserError("");
              }}
            >
              close
            </button>
          </form>
        </dialog>

        {tab === "signin" && (
          <div className="space-y-6">
            <p className="text-base-content/70">
              Configure how your users can authenticate with your application.
            </p>

            {configError && (
              <div className="alert alert-error text-sm">{configError}</div>
            )}

            {configLoading ? (
              <div className="flex justify-center py-12">
                <span className="loading loading-spinner loading-md" />
              </div>
            ) : (
              <div className="card bg-base-100 shadow-md">
                <div className="card-body">
                  <h2 className="card-title text-lg">
                    Authentication Providers
                  </h2>
                  <div className="overflow-x-auto">
                    <table className="table">
                      <thead>
                        <tr>
                          <th>Provider</th>
                          <th>Status</th>
                          <th>Actions</th>
                        </tr>
                      </thead>
                      <tbody>
                        {DEFAULT_PROVIDER_KEYS.map((key) => {
                          const meta = PROVIDER_META[key];
                          const providerConfig = (config.providers || {})[
                            key
                          ] as { enabled?: boolean } | undefined;
                          const enabled = providerConfig?.enabled === true;
                          const Icon = meta?.icon ?? Mail;
                          return (
                            <tr key={key}>
                              <td>
                                <div className="flex items-center gap-3">
                                  <div className="w-8 h-8 rounded-lg bg-primary/10 flex items-center justify-center">
                                    <Icon className="w-4 h-4 text-primary" />
                                  </div>
                                  <div>
                                    <p className="font-medium">
                                      {meta?.label ?? key}
                                    </p>
                                    <p className="text-sm text-base-content/60">
                                      {meta?.description ?? ""}
                                    </p>
                                  </div>
                                </div>
                              </td>
                              <td>
                                {enabled ? (
                                  <span className="badge badge-success gap-1">
                                    <span className="w-1.5 h-1.5 rounded-full bg-current" />
                                    Enabled
                                  </span>
                                ) : (
                                  <span className="badge badge-ghost gap-1">
                                    <span className="w-1.5 h-1.5 rounded-full bg-current opacity-60" />
                                    Disabled
                                  </span>
                                )}
                              </td>
                              <td>
                                {enabled ? (
                                  <button
                                    type="button"
                                    className="btn btn-ghost btn-sm"
                                    disabled={configSaving}
                                    onClick={() => toggleProvider(key, false)}
                                  >
                                    Edit
                                  </button>
                                ) : (
                                  <button
                                    type="button"
                                    className="btn btn-primary btn-sm"
                                    disabled={configSaving}
                                    onClick={() => toggleProvider(key, true)}
                                  >
                                    Enable
                                  </button>
                                )}
                              </td>
                            </tr>
                          );
                        })}
                      </tbody>
                    </table>
                  </div>
                  <div className="alert bg-base-200 border-base-300 mt-4">
                    <Info className="w-5 h-5 text-primary" />
                    <div>
                      <p className="font-medium">
                        Need help configuring providers?
                      </p>
                      <p className="text-sm text-base-content/70">
                        Check out our Authentication guides for step-by-step
                        instructions on setting up OAuth applications for
                        Google, GitHub, and more.
                      </p>
                    </div>
                  </div>
                </div>
              </div>
            )}
          </div>
        )}
      </main>
    </div>
  );
}
