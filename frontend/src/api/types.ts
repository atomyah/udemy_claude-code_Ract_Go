import type { components } from './generated/schema';

type Schemas = components['schemas'];

export type UserResponse = Schemas['github_com_atyahara_sns-backend_internal_dto.UserResponse'];
export type UserInPost = Schemas['github_com_atyahara_sns-backend_internal_dto.UserInPost'];
export type UserListResponse = Schemas['github_com_atyahara_sns-backend_internal_dto.UserListResponse'];

export type PostResponse = Schemas['github_com_atyahara_sns-backend_internal_dto.PostResponse'];
export type PostSummary = Schemas['github_com_atyahara_sns-backend_internal_dto.PostSummary'];
export type PostListResponse = Schemas['github_com_atyahara_sns-backend_internal_dto.PostListResponse'];
export type MediaResponse = Schemas['github_com_atyahara_sns-backend_internal_dto.MediaResponse'];
export type UpdatePostRequest = Schemas['github_com_atyahara_sns-backend_internal_dto.UpdatePostRequest'];

export type LikeResponse = Schemas['github_com_atyahara_sns-backend_internal_dto.LikeResponse'];
export type RepostResponse = Schemas['github_com_atyahara_sns-backend_internal_dto.RepostResponse'];

export type NotificationResponse = Schemas['github_com_atyahara_sns-backend_internal_dto.NotificationResponse'];
export type NotificationListResponse =
  Schemas['github_com_atyahara_sns-backend_internal_dto.NotificationListResponse'];

export type AuthResponse = Schemas['github_com_atyahara_sns-backend_internal_dto.AuthResponse'];
export type RefreshResponse = Schemas['github_com_atyahara_sns-backend_internal_dto.RefreshResponse'];
export type LoginRequest = Schemas['github_com_atyahara_sns-backend_internal_dto.LoginRequest'];
export type RegisterRequest = Schemas['github_com_atyahara_sns-backend_internal_dto.RegisterRequest'];
export type GoogleLoginRequest = Schemas['github_com_atyahara_sns-backend_internal_dto.GoogleLoginRequest'];

export type UpdateProfileRequest = Schemas['github_com_atyahara_sns-backend_internal_dto.UpdateProfileRequest'];
export type UpdateThemeRequest = Schemas['github_com_atyahara_sns-backend_internal_dto.UpdateThemeRequest'];
export type AvatarResponse = Schemas['github_com_atyahara_sns-backend_internal_dto.AvatarResponse'];
export type BannerResponse = Schemas['github_com_atyahara_sns-backend_internal_dto.BannerResponse'];
export type ChangeEmailRequest = Schemas['github_com_atyahara_sns-backend_internal_dto.ChangeEmailRequest'];
export type ChangePasswordRequest = Schemas['github_com_atyahara_sns-backend_internal_dto.ChangePasswordRequest'];
export type AdminUserResponse = Schemas['github_com_atyahara_sns-backend_internal_dto.AdminUserResponse'];
export type AdminUserListResponse = Schemas['github_com_atyahara_sns-backend_internal_dto.AdminUserListResponse'];

export type SuccessResponse = Schemas['github_com_atyahara_sns-backend_internal_dto.SuccessResponse'];
export type ErrorResponse = Schemas['github_com_atyahara_sns-backend_internal_dto.ErrorResponse'];
