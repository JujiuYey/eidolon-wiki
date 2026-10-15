import { z } from "zod";

// WeKnora 密码策略（internal/application/service/password_policy.go）：
// 恒定 8-32 位；auth.complex_password_enabled 开启时另需四类字符。
// 超管建号/重置与个人改密共用同一策略。

export function passwordHintText(complex: boolean): string {
  return complex ? "长度 8-32 位，需包含大写字母、小写字母、数字和特殊字符" : "长度 8-32 位";
}

export function buildPasswordSchema(complex: boolean) {
  let schema = z
    .string()
    .min(8, "密码长度至少为 8 位")
    .max(32, "密码长度不能超过 32 位");
  if (complex) {
    schema = schema
      .regex(/[A-Z]/, "需包含大写字母")
      .regex(/[a-z]/, "需包含小写字母")
      .regex(/[0-9]/, "需包含数字")
      .regex(/[^A-Za-z0-9]/, "需包含特殊字符");
  }
  return schema;
}
