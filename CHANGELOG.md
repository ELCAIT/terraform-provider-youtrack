## 1.0.3
FEATURES:

IMPROVEMENTS:
- Remove mandatory field_type parameter for project custom field resource. It is now derived from the global custom field type.

BUG FIXES:
- Fix error when removing a bundle value.

## 1.0.2
FEATURES:

IMPROVEMENTS:
- Update dependencies.

BUG FIXES:
- Fix clearing of customfield empty_field_text attribute.
- Fix notification settings acceptance test when mail server not configured in the youtrack instance.

## 1.0.1
FEATURES:

IMPROVEMENTS:
- Changed licence from GPL-3 to MPL-2.0

BUG FIXES:

## 1.0.0
FEATURES:

IMPROVEMENTS:
- Refactor codebase to be public
- Extract client to a dedicated public go library (youtrack-api-client)

BUG FIXES:

## 0.5.0
FEATURES:
- Add support for issue link type
- Add support for customfields
- Add support for Enum and State bundles (templates for custom fields that can be used in multiple projects)
- Add support for project management (to create project templates)

IMPROVEMENTS:

BUG FIXES:

## 0.4.0
FEATURES:
- Add support for Time Tracking settings, allowing users to manage time tracking related configurations such as work item types, time tracking modes, and other related settings.
- Add support for Youtrack OAuth2 authentication module, allowing users to manage OAuth2 auth modules and their configurations.

IMPROVEMENTS:
- Refined github copilot instructions.
- Refactor codebase to improve maintainability and readability, including splitting large functions into smaller ones, improving error handling, and adding more comments for clarity.
- Add integration tests to cover the new features and ensure the stability of the provider.
- Update dependencies.

BUG FIXES:
- Fix role management after upgrade to youtrack 2026.1, which split role management into HUB and youtrack API.
- Fix acceptance tests

## 0.3.0
FEATURES:
- Add support for role Assignment only for global scope. Project-specific role assignments are not supported.
- Add support for database backup settings.

## 0.2.1
IMPROVEMENTS:
- Use artifactory as distribution platform for the provider, allowing users to easily download and install the provider from a central repository.
- Update dependencies.

BUG FIXES:
- Fix license management for youtrack 2026.1

## 0.2.0
FEATURES:
- Add system settings resource to manage global settings of YouTrack, including license information, appearance settings, and other global configurations.
- Implement REST API settings resource to manage REST API related settings, such as allowed origins and CORS configuration.

IMPROVEMENTS:
- Refactor codebase to improve maintainability and readability, including splitting large functions into smaller ones, improving error handling, and adding more comments for clarity.
- Add more unit tests to cover edge cases and ensure the stability of the provider.
- Update documentation to reflect the new resources and provide clear examples for users.

## 0.1.1
BUG FIXES:
- Role management: Fix new role creation. The new role was considered as updated instead of created.
- Mail server: Remove unneeded multiple mail server management as there is only one mail server configuration in YouTrack.
- Fix various code smells
- Fix cyclomatic complexity in role management by splitting the code into smaller functions and improving error handling.

IMPROVEMENTS:
- Role management: switch from a resource per role instead of a single resource for all roles. This allows to manage each role independently and avoid issues with concurrent updates. This allows single role import in the state.
- Add copilot instruction to ensure that the generated code is clean, maintenable and follows best practices for Terraform provider development.
- Add more unit tests to cover edge cases and ensure the stability of the provider.
- Update documentation to reflect the changes in the provider and provide clear examples for users.

## 0.1.0

FEATURES:
- Initial provider implementation
- Add provider schema with base_url and token configuration
- Implement youtrack_role resource with create, read, update, delete operations
- Implement youtrack_mail_server resource with create, read, update, delete operations
