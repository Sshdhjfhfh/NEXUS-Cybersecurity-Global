import packageJson from "../../package.json";

const currentYear = new Date().getFullYear();

export const APP_CONFIG = {
  name: "Nexus IA",
  version: packageJson.version,
  copyright: `© ${currentYear}, Nexus IA Project.`,
  meta: {
    title: "Nexus IA — Consola de Investigación Adversarial Autónoma",
    description: "Sistema de Investigación Adversarial Autónomo con Gobernanza Zero-Trust",
  },
};
