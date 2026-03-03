import { z } from "zod";

const isoDateTimeSchema = z.string().refine((value) => !Number.isNaN(Date.parse(value)), {
  message: "Expected an ISO-8601 datetime string",
});

export const checkStatusSchema = z.enum(["up", "late", "down", "paused"]);
export const alertChannelTypeSchema = z.enum(["email", "webhook"]);
export const scheduleModeSchema = z.enum(["manual", "auto"]);

export const errorResponseSchema = z.object({
  code: z.string(),
  message: z.string(),
});

export const pingBodySchema = z
  .object({
    duration_ms: z.number().int().nonnegative().optional(),
    output_size: z.number().int().nonnegative().optional(),
    success: z.boolean().optional(),
    state: z.enum(["run", "ok", "fail"]).optional(),
    msg: z.string().max(1024).optional(),
    env: z.string().max(64).optional(),
    metric: z.string().max(256).optional(),
  })
  .strict();

export const pingResponseSchema = z.object({
  ok: z.boolean(),
  status: checkStatusSchema,
  next_due_at: isoDateTimeSchema,
  idempotent: z.boolean().optional(),
});

export const checkSchema = z.object({
  id: z.string().uuid(),
  user_id: z.string().uuid(),
  name: z.string(),
  token: z.string(),
  expected_interval_seconds: z.number().int().positive(),
  grace_seconds: z.number().int().nonnegative(),
  schedule_mode: scheduleModeSchema.optional(),
  interval_sample_count: z.number().int().nonnegative().optional(),
  status: checkStatusSchema,
  last_ping_at: isoDateTimeSchema.nullable().optional(),
  next_due_at: isoDateTimeSchema.nullable().optional(),
  created_at: isoDateTimeSchema,
  updated_at: isoDateTimeSchema,
});

export const listChecksResponseSchema = z.object({
  items: z.array(checkSchema),
});

export const createCheckRequestSchema = z.object({
  name: z.string().trim().min(1),
  expected_interval_seconds: z.number().int().positive().optional(),
  grace_seconds: z.number().int().nonnegative().optional(),
  schedule_mode: scheduleModeSchema.optional(),
});

export const patchCheckRequestSchema = z
  .object({
    name: z.string().trim().min(1).optional(),
    expected_interval_seconds: z.number().int().positive().optional(),
    grace_seconds: z.number().int().nonnegative().optional(),
    schedule_mode: scheduleModeSchema.optional(),
  })
  .refine(
    (value) =>
      value.name !== undefined ||
      value.expected_interval_seconds !== undefined ||
      value.grace_seconds !== undefined ||
      value.schedule_mode !== undefined,
    { message: "At least one field is required for patch" }
  );

export const eventSchema = z.object({
  id: z.number().int().nonnegative(),
  type: z.string(),
  received_at: isoDateTimeSchema,
  duration_ms: z.number().int().nullable().optional(),
  output_size: z.number().int().nullable().optional(),
  success: z.boolean().nullable().optional(),
  meta_json: z.string().nullable().optional(),
});

export const listEventsResponseSchema = z.object({
  items: z.array(eventSchema),
});

export const alertChannelSchema = z.object({
  id: z.string().uuid(),
  type: alertChannelTypeSchema,
  target: z.string(),
  enabled: z.boolean(),
  created_at: isoDateTimeSchema,
});

export const listAlertChannelsResponseSchema = z.object({
  items: z.array(alertChannelSchema),
});

export const createAlertChannelRequestSchema = z.object({
  type: alertChannelTypeSchema,
  target: z.string().trim().min(1),
});

export const patchAlertChannelRequestSchema = z
  .object({
    enabled: z.boolean().optional(),
    target: z.string().trim().min(1).optional(),
  })
  .refine((value) => value.enabled !== undefined || value.target !== undefined, {
    message: "At least one field is required for patch",
  });

export type ErrorResponse = z.infer<typeof errorResponseSchema>;
export type PingBody = z.infer<typeof pingBodySchema>;
export type PingResponse = z.infer<typeof pingResponseSchema>;
export type Check = z.infer<typeof checkSchema>;
export type ListChecksResponse = z.infer<typeof listChecksResponseSchema>;
export type CreateCheckRequest = z.infer<typeof createCheckRequestSchema>;
export type PatchCheckRequest = z.infer<typeof patchCheckRequestSchema>;
export type Event = z.infer<typeof eventSchema>;
export type ListEventsResponse = z.infer<typeof listEventsResponseSchema>;
export type AlertChannel = z.infer<typeof alertChannelSchema>;
export type ListAlertChannelsResponse = z.infer<typeof listAlertChannelsResponseSchema>;
export type CreateAlertChannelRequest = z.infer<typeof createAlertChannelRequestSchema>;
export type PatchAlertChannelRequest = z.infer<typeof patchAlertChannelRequestSchema>;
