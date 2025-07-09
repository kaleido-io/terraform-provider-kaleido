terraform {
  required_providers {
    kaleido = {
      source  = "kaleido-io/kaleido"
      version = "~> 1.0"
    }
  }
}

provider "kaleido" {
  # Configuration will be provided via environment variables or configuration files
  platform_api = var.platform_api_url
  platform_username = var.platform_api_key_name
  platform_password = var.platform_api_key_value
}

variable "platform_api_url" {
  description = "URL of the platform API"
  type        = string
}

variable "platform_api_key_name" {
  description = "Name of the platform API key"
}

variable "platform_api_key_value" {
  description = "Value of the platform API key"
}

variable "groups" {
  description = "Groups to create within the account"
  type = list(object({
    name    = string
    description = optional(string)
  }))
  default = [
    {
      name    = "developers"
      description = "allow read, write on al resources"
    },
    {
      name    = "admins"
      description = "allow all operations on all resources"
    },
    {
      name    = "viewers"
      description = "allow read on all resources"
    }
  ]
}

variable "users" {
  description = "Users to create within the account"
  type = list(object({
    name     = string
    email    = optional(string)
    sub      = optional(string)
    is_admin = optional(bool, false)
    groups   = list(string)  # List of group names this user should belong to
  }))
  default = [
    {
      name     = "john.doe"
      email    = "john.doe@example.com"
      sub      = "john.doe.sub"
      is_admin = false
      groups   = ["developers", "viewers"]
    },
    {
      name     = "jane.smith"
      email    = "jane.smith@example.com"
      sub      = "jane.smith.sub"
      is_admin = true
      groups   = ["admins"]
    },
    {
      name     = "bob.wilson"
      email    = "bob.wilson@example.com"
      sub      = "bob.wilson.sub"
      is_admin = false
      groups   = ["developers"]
    },
    {
      name     = "alice.brown"
      email    = "alice.brown@example.com"
      sub      = "alice.brown.sub"
      is_admin = false
      groups   = ["viewers"]
    },
    {
      name     = "service-account"
      email    = "service@example.com"
      sub      = "service.account.sub"
      is_admin = false
      groups   = ["developers"]
    }
  ]
}

# Create groups within the account
resource "kaleido_platform_group" "groups" {
  for_each = {
    for group in var.groups : group.name => group
  }
  
  name    = each.value.name
}

data "kaleido_platform_account" "current" {}

# Create users within the account
resource "kaleido_platform_user" "users" {
  for_each = {
    for user in var.users : user.name => user
  }
  
  name       = each.value.name
  email      = each.value.email
  sub        = each.value.sub
  is_admin   = each.value.is_admin
}

# Create group memberships for users
resource "kaleido_platform_group_membership" "user_group_memberships" {
  for_each = {
    for membership in flatten([
      for user in var.users : [
        for group_name in user.groups : {
          user_name  = user.name
          group_name = group_name
          key        = "${user.name}-${group_name}"
        }
      ]
    ]) : membership.key => membership
  }
  
  group_id = kaleido_platform_group.groups[each.value.group_name].id
  user_id  = kaleido_platform_user.users[each.value.user_name].id
}

# Local values for organization
locals {
  # Create a mapping of groups to their members
  group_members = {
    for group_name in keys(kaleido_platform_group.groups) : group_name => [
      for user in var.users : user.name
      if contains(user.groups, group_name)
    ]
  }
  
  # Create a mapping of users to their group memberships
  user_groups = {
    for user in var.users : user.name => user.groups
  }
  
  # Separate admin and non-admin users
  admin_users = [
    for user in var.users : user.name
    if user.is_admin
  ]
  
  non_admin_users = [
    for user in var.users : user.name
    if !user.is_admin
  ]
}

# Output the created groups
output "groups" {
  description = "Details of created groups"
  value = {
    for name, group in kaleido_platform_group.groups : name => {
      id       = group.id
      name     = group.name
      members  = local.group_members[name]
    }
  }
}

# Output the created users
output "users" {
  description = "Details of created users"
  value = {
    for name, user in kaleido_platform_user.users : name => {
      id         = user.id
      name       = user.name
      email      = user.email
      is_admin   = user.is_admin
      account = user.account
      groups     = local.user_groups[name]
    }
  }
}

# Output group memberships
output "group_memberships" {
  description = "Details of group memberships"
  value = {
    for key, membership in kaleido_platform_group_membership.user_group_memberships : key => {
      id         = membership.id
      group_id   = membership.group_id
      user_id    = membership.user_id
      group_name = membership.group_name
      user_name  = membership.user_name
      account_id = membership.account_id
    }
  }
}

# Output summary statistics
output "iam_summary" {
  description = "Summary of IAM configuration"
  value = {
    account         = data.kaleido_platform_account.current.account_id
    total_groups       = length(kaleido_platform_group.groups)
    total_users        = length(kaleido_platform_user.users)
    total_memberships  = length(kaleido_platform_group_membership.user_group_memberships)
    admin_users        = local.admin_users
    non_admin_users    = local.non_admin_users
    group_names        = [for group in kaleido_platform_group.groups : group.name]
    user_names         = [for user in kaleido_platform_user.users : user.name]
  }
}

# Output organized by access level
output "access_organization" {
  description = "Users organized by access level"
  value = {
    administrators = {
      users = [
        for user_name in local.admin_users : {
          name   = user_name
          email  = kaleido_platform_user.users[user_name].email
          groups = local.user_groups[user_name]
        }
      ]
    }
    developers = {
      users = [
        for user_name in local.non_admin_users : {
          name   = user_name
          email  = kaleido_platform_user.users[user_name].email
          groups = local.user_groups[user_name]
        }
        if contains(local.user_groups[user_name], "developers")
      ]
    }
    readonly_users = {
      users = [
        for user_name in local.non_admin_users : {
          name   = user_name
          email  = kaleido_platform_user.users[user_name].email
          groups = local.user_groups[user_name]
        }
        if contains(local.user_groups[user_name], "readonly")
      ]
    }
    api_users = {
      users = [
        for user_name in local.non_admin_users : {
          name   = user_name
          email  = kaleido_platform_user.users[user_name].email
          groups = local.user_groups[user_name]
        }
        if contains(local.user_groups[user_name], "api-users")
      ]
    }
  }
} 