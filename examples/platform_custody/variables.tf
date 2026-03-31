variable "environment_name" {
  type = string
}

variable "databases" {
  type = object({
    wfe_db           = string
    cms_db           = string
    kms_db           = string
    bis_db           = string
    evmconnector_db  = string
  })
  default = null
}

variable "kaleido_platform_api" {
  type = string
}

variable "kaleido_platform_username" {
  type = string
}

variable "kaleido_platform_password" {
  type = string
}