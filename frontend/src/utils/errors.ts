import { ApiError } from "../api/client";
import type { MessageKey } from "../i18n";

type Translate = (key: MessageKey, params?: Record<string, string | number>) => string;

export function apiErrorMessage(error: unknown, t: Translate): string {
  if (error instanceof ApiError) {
    switch (error.code) {
      case "company_required":
        return t("error.companyRequired");
      case "invalid_category":
        return t("error.invalidCategory");
      case "invalid_status":
        return t("error.invalidStatus");
      case "too_many_ids":
        return t("error.tooMany");
      case "invalid_password":
        return t("error.passwordIncorrect");
      case "network_error":
        return t("error.network");
      case "unauthorized":
        return t("error.unauthorized");
      default:
        return error.message || t("error.generic");
    }
  }
  return t("error.generic");
}
