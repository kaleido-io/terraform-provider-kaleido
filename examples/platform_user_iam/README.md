# Platform User IAM Example

This example demonstrates how to manage users, groups, and group memberships within a Kaleido platform account using Terraform. It shows how to implement a comprehensive Identity and Access Management (IAM) system for organizing users and controlling access to platform resources.

## Overview

This configuration creates:
- Multiple user groups with different access levels
- Individual users with specific roles and permissions
- Group memberships that assign users to appropriate groups
- Comprehensive output for review

## Prerequisites

Before running this example, ensure you have:
1. OpenTofu installed (version 1.9 or later)
2. Access to a Kaleido platform instance
3. An existing account ID where users and groups will be created
4. Appropriate administrative credentials for the Kaleido platform

## Configuration

### Required Variables

You must provide the account ID where users and groups will be managed:

```hcl
account_id = "your-account-id"
```

### Optional Variables

You can customize the groups and users by modifying the default variables:

#### Groups Configuration
```hcl
groups = [
  {
    name    = "developers"
  },
  {
    name    = "admins"
  },
  # ... more groups
]
```

#### Users Configuration
```hcl
users = [
  {
    name     = "john.doe"
    email    = "john.doe@example.com"
    sub      = "john.doe.sub"
    is_admin = false
    groups   = ["developers", "readonly"]
  },
  # ... more users
]
```

## Usage

1. **Set up your variables** - Create a `input.tfvars` file:

```hcl
account_id = "your-account-id"

# Optional: Override default groups
groups = [
  {
    name    = "backend-devs"
    ruleset = "allow read, write on backend services"
  },
  {
    name    = "frontend-devs"
    ruleset = "allow read, write on frontend services"
  }
]

# Optional: Override default users
users = [
  {
    name     = "alice.developer"
    email    = "alice@company.com"
    sub      = "alice.developer.sub"
    is_admin = false
    groups   = ["backend-devs"]
  }
]
```

2. **Initialize Terraform:**

```bash
tofu init
```

3. **Plan the deployment:**

```bash
terraform plan -var-file=input.tfvars 
```

4. **Apply the configuration:**

```bash
terraform apply -var-file=input.tfvars 
```

## Resources Created

### Groups
- **Type**: `kaleido_platform_group`
- **Purpose**: Organize users by role and access level
- **Default Groups**:
  - `developers`: Development team members
  - `admins`: Platform administrators
  - `readonly`: Read-only access users
  - `api-users`: API service accounts

### Users
- **Type**: `kaleido_platform_user`
- **Purpose**: Individual user accounts within the platform
- **Features**:
  - Email and OAuth subject identifier support
  - Admin privilege configuration
  - Account-scoped access

### Group Memberships
- **Type**: `kaleido_platform_group_membership`
- **Purpose**: Assign users to groups for access control
- **Features**:
  - Many-to-many relationship support
  - Automatic membership management
  - Audit trail for access changes

## Key Features

### Role-Based Access Control
- **Hierarchical Permissions**: Admin users have elevated privileges
- **Group-Based Access**: Users inherit permissions from groups
- **Flexible Rulesets**: Custom rules for each group

### Multi-User Management
- **Batch Operations**: Create multiple users and groups simultaneously
- **Relationship Management**: Automatic group membership assignment
- **Audit Capabilities**: Comprehensive output for access monitoring

### Operational Flexibility
- **Service Accounts**: Support for non-human users
- **Dynamic Membership**: Easy to modify group assignments
- **Scalable Design**: Handles large numbers of users and groups

## Outputs

The configuration provides detailed outputs for monitoring and integration:

### Group Details
```hcl
output "groups" {
  # Complete group information including members
}
```

### User Details
```hcl
output "users" {
  # User information with group memberships
}
```

### Group Memberships
```hcl
output "group_memberships" {
  # Detailed membership relationships
}
```

### IAM Summary
```hcl
output "iam_summary" {
  # High-level statistics and organization
}
```

### Access Organization
```hcl
output "access_organization" {
  # Users organized by access level and role
}
```

## Access Patterns

### Administrative Access
- **Admin Users**: Full platform privileges
- **Admin Group**: Specialized administrative group membership
- **Elevated Permissions**: Can manage other users and groups

### Development Access
- **Developer Groups**: Environment-specific access
- **Read/Write Permissions**: Development resource access
- **Collaborative Features**: Team-based access patterns

### Read-Only Access
- **Monitoring Users**: View-only access to resources
- **Audit Accounts**: Access for compliance and monitoring
- **Restricted Permissions**: Limited to read operations

### API Access
- **Service Accounts**: Non-human user access
- **API-Specific Groups**: Tailored for service integration
- **Automated Access**: Support for CI/CD and automation

## Operational Considerations

### User Lifecycle Management
- **Onboarding**: Easy addition of new users
- **Role Changes**: Simple group membership updates
- **Offboarding**: Clean removal of user access

### Security
- **Least Privilege**: Users get minimum required access
- **Group-Based Security**: Consistent permission application
- **Audit Trail**: Complete access history tracking

### Scalability
- **Batch Operations**: Handle many users efficiently
- **Dynamic Configuration**: Easy to add new roles and users
- **Performance**: Optimized for large user bases

## Advanced Configuration

### Custom Group Rules
```hcl
variable "custom_groups" {
  type = list(object({
    name    = string
    ruleset = string
  }))
  default = [
    {
      name    = "data-scientists"
      ruleset = "allow read on data resources, write on analytics"
    },
    {
      name    = "security-team"
      ruleset = "allow read on all resources, write on security configs"
    }
  ]
}
```

### External User Integration
```hcl
variable "external_users" {
  type = list(object({
    name     = string
    email    = string
    sub      = string
    is_admin = bool
    groups   = list(string)
  }))
  # Load from external source like LDAP, Microsoft Entra ID, etc.
}
```

## Troubleshooting

### Common Issues

1. **User Creation Failures**:
   - Verify account ID exists and is accessible
   - Check user email format and uniqueness
   - Ensure OAuth subject identifiers are valid

2. **Group Assignment Problems**:
   - Verify group names match exactly
   - Check for circular dependencies
   - Ensure proper user and group creation order

3. **Permission Issues**:
   - Verify admin privileges for user management
   - Check account-level permissions
   - Ensure proper API authentication

### Validation Commands

```bash
# Check user creation
tofu output users

# Verify group memberships
tofu output group_memberships

# Review access organization
tofu output access_organization
```

## Integration Examples

### With Other Platform Resources
```hcl
module "user_iam" {
  source = "./platform_user_iam"
  
  account_id = kaleido_platform_account.my_account.id
}

# Use groups in other configurations
resource "kaleido_platform_service_access" "developer_access" {
  service_id = kaleido_platform_service.my_service.id
  group_id   = module.user_iam.groups["developers"].id
}
```

### With External Systems
```hcl
# Export user data for external systems
locals {
  user_export = {
    for user in module.user_iam.users : user.name => {
      email  = user.email
      groups = user.groups
      admin  = user.is_admin
    }
  }
}
```

## Security Best Practices

1. **Regular Access Reviews**: Periodically review user access and group memberships
2. **Principle of Least Privilege**: Grant minimum required access to users
3. **Separation of Duties**: Use different groups for different responsibilities
4. **Automated Compliance**: Use Terraform for consistent access management

## Clean Up

To remove all IAM resources:

```bash
tofu destroy
```

**Warning**: This will remove all users, groups, and memberships. Ensure you have proper backups and that no critical access will be lost.