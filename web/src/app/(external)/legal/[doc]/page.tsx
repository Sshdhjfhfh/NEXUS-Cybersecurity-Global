import { LEGAL_ORDER } from "../legal-content";
import { LegalDocView } from "./legal-doc-view";

// Every legal document is known at build time: the static export (the
// production build embedded in the backend) needs the list, and any other
// /legal/<x> is a 404.
export const dynamicParams = false;

export function generateStaticParams() {
  return LEGAL_ORDER.map((doc) => ({ doc }));
}

export default function LegalDocPage() {
  return <LegalDocView />;
}
