/**
 * Organizations Module
 *
 * Organization admin management.
 * Re-exports Organizations page and components.
 */

export { default as OrganizationDetail } from "./pages/OrganizationDetail";
export { default as Organizations } from "./pages/Organizations";
// NOTE: OrganizationUsers is in @modules/users

// Components re-exports
export { default as AddOrganizationDialog } from "./components/AddOrganizationDialog";
// NOTE: AddOrgUserDialog replaced by OrganizationUserDialog in @modules/users
export { default as OrganizationsTable } from "./components/OrganizationsTable";
export { default as TagsEditor } from "./components/TagsEditor";
