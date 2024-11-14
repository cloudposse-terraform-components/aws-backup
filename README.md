<!-- markdownlint-disable -->
<a href="https://cpco.io/homepage"><img src=".github/banner.png?raw=true" alt="Project Banner"/></a><br/>
    <p align="right">
<a href="https://github.com/cloudposse-terraform-components/aws-backup/releases/latest"><img src="https://img.shields.io/github/release/cloudposse-terraform-components/aws-backup.svg?style=for-the-badge" alt="Latest Release"/></a><a href="https://slack.cloudposse.com"><img src="https://slack.cloudposse.com/for-the-badge.svg" alt="Slack Community"/></a></p>
<!-- markdownlint-restore -->

<!--




  ** DO NOT EDIT THIS FILE
  **
  ** This file was automatically generated by the `cloudposse/build-harness`.
  ** 1) Make all changes to `README.yaml`
  ** 2) Run `make init` (you only need to do this once)
  ** 3) Run`make readme` to rebuild this file.
  **
  ** (We maintain HUNDREDS of open source projects. This is how we maintain our sanity.)
  **





-->

This component is responsible for provisioning an AWS Backup Plan. It creates a schedule for backing up given ARNs.

## Usage

**Stack Level**: Regional

Here's an example snippet for how to use this component.

### Component Abstraction and Separation

By separating the "common" settings from the component, we can first provision the IAM Role and AWS Backup Vault to
prepare resources for future use without incuring cost.

For example, `stacks/catalog/aws-backup/common`:

```yaml
# This configuration creates the AWS Backup Vault and IAM Role, and does not incur any cost on its own.
# See: https://aws.amazon.com/backup/pricing/
components:
  terraform:
    aws-backup:
      metadata:
        type: abstract
      settings:
        spacelift:
          workspace_enabled: true
      vars: {}

    aws-backup/common:
      metadata:
        component: aws-backup
        inherits:
          - aws-backup
      vars:
        enabled: true
        iam_role_enabled: true # this will be reused
        vault_enabled: true # this will be reused
        plan_enabled: false
## Please be careful when enabling backup_vault_lock_configuration,
#        backup_vault_lock_configuration:
##         `changeable_for_days` enables compliance mode and once the lock is set, the retention policy cannot be changed unless through account deletion!
#          changeable_for_days: 36500
#          max_retention_days: 365
#          min_retention_days: 1
```

Then if we would like to deploy the component into a given stacks we can import the following to deploy our backup
plans.

Since most of these values are shared and common, we can put them in a `catalog/aws-backup/` yaml file and share them
across environments.

This makes deploying the same configuration to multiple environments easy.

`stacks/catalog/aws-backup/defaults`:

```yaml
import:
  - catalog/aws-backup/common

components:
  terraform:
    aws-backup/plan-defaults:
      metadata:
        component: aws-backup
        type: abstract
      settings:
        spacelift:
          workspace_enabled: true
        depends_on:
          - aws-backup/common
      vars:
        enabled: true
        iam_role_enabled: false # reuse from aws-backup-vault
        vault_enabled: false # reuse from aws-backup-vault
        plan_enabled: true
        plan_name_suffix: aws-backup-defaults

    aws-backup/daily-plan:
      metadata:
        component: aws-backup
        inherits:
          - aws-backup/plan-defaults
      vars:
        plan_name_suffix: aws-backup-daily
        # https://docs.aws.amazon.com/AmazonCloudWatch/latest/events/ScheduledEvents.html
        rules:
          - name: "plan-daily"
            schedule: "cron(0 5 ? * * *)"
            start_window: 320 # 60 * 8             # minutes
            completion_window: 10080 # 60 * 24 * 7 # minutes
            lifecycle:
              delete_after: 35 # 7 * 5               # days
        selection_tags:
          - type: STRINGEQUALS
            key: aws-backup/efs
            value: daily
          - type: STRINGEQUALS
            key: aws-backup/rds
            value: daily

    aws-backup/weekly-plan:
      metadata:
        component: aws-backup
        inherits:
          - aws-backup/plan-defaults
      vars:
        plan_name_suffix: aws-backup-weekly
        # https://docs.aws.amazon.com/AmazonCloudWatch/latest/events/ScheduledEvents.html
        rules:
          - name: "plan-weekly"
            schedule: "cron(0 5 ? * SAT *)"
            start_window: 320 # 60 * 8              # minutes
            completion_window: 10080 # 60 * 24 * 7  # minutes
            lifecycle:
              delete_after: 90 # 30 * 3               # days
        selection_tags:
          - type: STRINGEQUALS
            key: aws-backup/efs
            value: weekly
          - type: STRINGEQUALS
            key: aws-backup/rds
            value: weekly

    aws-backup/monthly-plan:
      metadata:
        component: aws-backup
        inherits:
          - aws-backup/plan-defaults
      vars:
        plan_name_suffix: aws-backup-monthly
        # https://docs.aws.amazon.com/AmazonCloudWatch/latest/events/ScheduledEvents.html
        rules:
          - name: "plan-monthly"
            schedule: "cron(0 5 1 * ? *)"
            start_window: 320 # 60 * 8              # minutes
            completion_window: 10080 # 60 * 24 * 7  # minutes
            lifecycle:
              delete_after: 2555 # 365 * 7            # days
              cold_storage_after: 90 # 30 * 3         # days
        selection_tags:
          - type: STRINGEQUALS
            key: aws-backup/efs
            value: monthly
          - type: STRINGEQUALS
            key: aws-backup/rds
            value: monthly
```

Deploying to a new stack (environment) then only requires:

```yaml
import:
  - catalog/aws-backup/defaults
```

The above configuration can be used to deploy a new backup to a new region.

---

### Adding Resources to the Backup - Adding Tags

Once an `aws-backup` with a plan and `selection_tags` has been established we can begin adding resources for it to
backup by using the tagging method.

This only requires that we add tags to the resources we wish to backup, which can be done with the following snippet:

```yaml
components:
  terraform:
    <my-resource>
      vars:
        tags:
          aws-backup/resource_schedule: "daily-14day-backup"
```

Just ensure the tag key-value pair matches what was added to your backup plan and aws will take care of the rest.

### Copying across regions

If we want to create a backup vault in another region that we can copy to, then we need to create another vault, and
then specify that we want to copy to it.

To create a vault in a region simply:

```yaml
components:
  terraform:
    aws-backup:
      vars:
        plan_enabled: false # disables the plan (which schedules resource backups)
```

This will output an ARN - which you can then use as the destination in the rule object's `copy_action` (it will be
specific to that particular plan), as seen in the following snippet:

```yaml
components:
  terraform:
    aws-backup/plan-with-cross-region-replication:
      metadata:
        component: aws-backup
        inherits:
          - aws-backup/plan-defaults
      vars:
        plan_name_suffix: aws-backup-cross-region
        # https://docs.aws.amazon.com/AmazonCloudWatch/latest/events/ScheduledEvents.html
        rules:
          - name: "plan-cross-region"
            schedule: "cron(0 5 ? * * *)"
            start_window: 320 # 60 * 8             # minutes
            completion_window: 10080 # 60 * 24 * 7 # minutes
            lifecycle:
              delete_after: 35 # 7 * 5               # days
            copy_action:
              destination_vault_arn: "arn:aws:backup:<other-region>:111111111111:backup-vault:<namespace>-<other-region>-<stage>"
              lifecycle:
                delete_after: 35
```

### Backup Lock Configuration

To enable backup lock configuration, you can use the following snippet:

- [AWS Backup Vault Lock](https://docs.aws.amazon.com/aws-backup/latest/devguide/vault-lock.html)

#### Compliance Mode

Vaults locked in compliance mode cannot be deleted once the cooling-off period ("grace time") expires. During grace
time, you can still remove the vault lock and change the lock configuration.

To enable **Compliance Mode**, set `changeable_for_days` to a value greater than 0. Once the lock is set, the retention
policy cannot be changed unless through account deletion!

```yaml
# Please be careful when enabling backup_vault_lock_configuration,
backup_vault_lock_configuration:
  #         `changeable_for_days` enables compliance mode and once the lock is set, the retention policy cannot be changed unless through account deletion!
  changeable_for_days: 36500
  max_retention_days: 365
  min_retention_days: 1
```

#### Governance Mode

Vaults locked in governance mode can have the lock removed by users with sufficient IAM permissions.

To enable **governance mode**

```yaml
backup_vault_lock_configuration:
  max_retention_days: 365
  min_retention_days: 1
```

<!-- prettier-ignore-start -->
<!-- BEGINNING OF PRE-COMMIT-TERRAFORM DOCS HOOK -->
## Requirements

| Name | Version |
|------|---------|
| <a name="requirement_terraform"></a> [terraform](#requirement\_terraform) | >= 1.3.0 |
| <a name="requirement_aws"></a> [aws](#requirement\_aws) | >= 4.9.0 |

## Providers

No providers.

## Modules

| Name | Source | Version |
|------|--------|---------|
| <a name="module_backup"></a> [backup](#module\_backup) | cloudposse/backup/aws | 1.0.0 |
| <a name="module_iam_roles"></a> [iam\_roles](#module\_iam\_roles) | ../account-map/modules/iam-roles | n/a |
| <a name="module_this"></a> [this](#module\_this) | cloudposse/label/null | 0.25.0 |

## Resources

No resources.

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_additional_tag_map"></a> [additional\_tag\_map](#input\_additional\_tag\_map) | Additional key-value pairs to add to each map in `tags_as_list_of_maps`. Not added to `tags` or `id`.<br>This is for some rare cases where resources want additional configuration of tags<br>and therefore take a list of maps with tag key, value, and additional configuration. | `map(string)` | `{}` | no |
| <a name="input_advanced_backup_setting"></a> [advanced\_backup\_setting](#input\_advanced\_backup\_setting) | An object that specifies backup options for each resource type. | <pre>object({<br>    backup_options = string<br>    resource_type  = string<br>  })</pre> | `null` | no |
| <a name="input_attributes"></a> [attributes](#input\_attributes) | ID element. Additional attributes (e.g. `workers` or `cluster`) to add to `id`,<br>in the order they appear in the list. New attributes are appended to the<br>end of the list. The elements of the list are joined by the `delimiter`<br>and treated as a single ID element. | `list(string)` | `[]` | no |
| <a name="input_backup_resources"></a> [backup\_resources](#input\_backup\_resources) | An array of strings that either contain Amazon Resource Names (ARNs) or match patterns of resources to assign to a backup plan | `list(string)` | `[]` | no |
| <a name="input_backup_vault_lock_configuration"></a> [backup\_vault\_lock\_configuration](#input\_backup\_vault\_lock\_configuration) | The backup vault lock configuration, each vault can have one vault lock in place. This will enable Backup Vault Lock on an AWS Backup vault  it prevents the deletion of backup data for the specified retention period. During this time, the backup data remains immutable and cannot be deleted or modified."<br>`changeable_for_days` - The number of days before the lock date. If omitted creates a vault lock in `governance` mode, otherwise it will create a vault lock in `compliance` mode. | <pre>object({<br>    changeable_for_days = optional(number)<br>    max_retention_days  = optional(number)<br>    min_retention_days  = optional(number)<br>  })</pre> | `null` | no |
| <a name="input_context"></a> [context](#input\_context) | Single object for setting entire context at once.<br>See description of individual variables for details.<br>Leave string and numeric variables as `null` to use default value.<br>Individual variable settings (non-null) override settings in context object,<br>except for attributes, tags, and additional\_tag\_map, which are merged. | `any` | <pre>{<br>  "additional_tag_map": {},<br>  "attributes": [],<br>  "delimiter": null,<br>  "descriptor_formats": {},<br>  "enabled": true,<br>  "environment": null,<br>  "id_length_limit": null,<br>  "label_key_case": null,<br>  "label_order": [],<br>  "label_value_case": null,<br>  "labels_as_tags": [<br>    "unset"<br>  ],<br>  "name": null,<br>  "namespace": null,<br>  "regex_replace_chars": null,<br>  "stage": null,<br>  "tags": {},<br>  "tenant": null<br>}</pre> | no |
| <a name="input_delimiter"></a> [delimiter](#input\_delimiter) | Delimiter to be used between ID elements.<br>Defaults to `-` (hyphen). Set to `""` to use no delimiter at all. | `string` | `null` | no |
| <a name="input_descriptor_formats"></a> [descriptor\_formats](#input\_descriptor\_formats) | Describe additional descriptors to be output in the `descriptors` output map.<br>Map of maps. Keys are names of descriptors. Values are maps of the form<br>`{<br>   format = string<br>   labels = list(string)<br>}`<br>(Type is `any` so the map values can later be enhanced to provide additional options.)<br>`format` is a Terraform format string to be passed to the `format()` function.<br>`labels` is a list of labels, in order, to pass to `format()` function.<br>Label values will be normalized before being passed to `format()` so they will be<br>identical to how they appear in `id`.<br>Default is `{}` (`descriptors` output will be empty). | `any` | `{}` | no |
| <a name="input_enabled"></a> [enabled](#input\_enabled) | Set to false to prevent the module from creating any resources | `bool` | `null` | no |
| <a name="input_environment"></a> [environment](#input\_environment) | ID element. Usually used for region e.g. 'uw2', 'us-west-2', OR role 'prod', 'staging', 'dev', 'UAT' | `string` | `null` | no |
| <a name="input_iam_role_enabled"></a> [iam\_role\_enabled](#input\_iam\_role\_enabled) | Whether or not to create a new IAM Role and Policy Attachment | `bool` | `true` | no |
| <a name="input_id_length_limit"></a> [id\_length\_limit](#input\_id\_length\_limit) | Limit `id` to this many characters (minimum 6).<br>Set to `0` for unlimited length.<br>Set to `null` for keep the existing setting, which defaults to `0`.<br>Does not affect `id_full`. | `number` | `null` | no |
| <a name="input_kms_key_arn"></a> [kms\_key\_arn](#input\_kms\_key\_arn) | The server-side encryption key that is used to protect your backups | `string` | `null` | no |
| <a name="input_label_key_case"></a> [label\_key\_case](#input\_label\_key\_case) | Controls the letter case of the `tags` keys (label names) for tags generated by this module.<br>Does not affect keys of tags passed in via the `tags` input.<br>Possible values: `lower`, `title`, `upper`.<br>Default value: `title`. | `string` | `null` | no |
| <a name="input_label_order"></a> [label\_order](#input\_label\_order) | The order in which the labels (ID elements) appear in the `id`.<br>Defaults to ["namespace", "environment", "stage", "name", "attributes"].<br>You can omit any of the 6 labels ("tenant" is the 6th), but at least one must be present. | `list(string)` | `null` | no |
| <a name="input_label_value_case"></a> [label\_value\_case](#input\_label\_value\_case) | Controls the letter case of ID elements (labels) as included in `id`,<br>set as tag values, and output by this module individually.<br>Does not affect values of tags passed in via the `tags` input.<br>Possible values: `lower`, `title`, `upper` and `none` (no transformation).<br>Set this to `title` and set `delimiter` to `""` to yield Pascal Case IDs.<br>Default value: `lower`. | `string` | `null` | no |
| <a name="input_labels_as_tags"></a> [labels\_as\_tags](#input\_labels\_as\_tags) | Set of labels (ID elements) to include as tags in the `tags` output.<br>Default is to include all labels.<br>Tags with empty values will not be included in the `tags` output.<br>Set to `[]` to suppress all generated tags.<br>**Notes:**<br>  The value of the `name` tag, if included, will be the `id`, not the `name`.<br>  Unlike other `null-label` inputs, the initial setting of `labels_as_tags` cannot be<br>  changed in later chained modules. Attempts to change it will be silently ignored. | `set(string)` | <pre>[<br>  "default"<br>]</pre> | no |
| <a name="input_name"></a> [name](#input\_name) | ID element. Usually the component or solution name, e.g. 'app' or 'jenkins'.<br>This is the only ID element not also included as a `tag`.<br>The "name" tag is set to the full `id` string. There is no tag with the value of the `name` input. | `string` | `null` | no |
| <a name="input_namespace"></a> [namespace](#input\_namespace) | ID element. Usually an abbreviation of your organization name, e.g. 'eg' or 'cp', to help ensure generated IDs are globally unique | `string` | `null` | no |
| <a name="input_plan_enabled"></a> [plan\_enabled](#input\_plan\_enabled) | Whether or not to create a new Plan | `bool` | `true` | no |
| <a name="input_plan_name_suffix"></a> [plan\_name\_suffix](#input\_plan\_name\_suffix) | The string appended to the plan name | `string` | `null` | no |
| <a name="input_regex_replace_chars"></a> [regex\_replace\_chars](#input\_regex\_replace\_chars) | Terraform regular expression (regex) string.<br>Characters matching the regex will be removed from the ID elements.<br>If not set, `"/[^a-zA-Z0-9-]/"` is used to remove all characters other than hyphens, letters and digits. | `string` | `null` | no |
| <a name="input_region"></a> [region](#input\_region) | AWS Region | `string` | n/a | yes |
| <a name="input_rules"></a> [rules](#input\_rules) | An array of rule maps used to define schedules in a backup plan | <pre>list(object({<br>    name                     = string<br>    schedule                 = optional(string)<br>    enable_continuous_backup = optional(bool)<br>    start_window             = optional(number)<br>    completion_window        = optional(number)<br>    lifecycle = optional(object({<br>      cold_storage_after                        = optional(number)<br>      delete_after                              = optional(number)<br>      opt_in_to_archive_for_supported_resources = optional(bool)<br>    }))<br>    copy_action = optional(object({<br>      destination_vault_arn = optional(string)<br>      lifecycle = optional(object({<br>        cold_storage_after                        = optional(number)<br>        delete_after                              = optional(number)<br>        opt_in_to_archive_for_supported_resources = optional(bool)<br>      }))<br>    }))<br>  }))</pre> | `[]` | no |
| <a name="input_selection_tags"></a> [selection\_tags](#input\_selection\_tags) | An array of tag condition objects used to filter resources based on tags for assigning to a backup plan | `list(map(string))` | `[]` | no |
| <a name="input_stage"></a> [stage](#input\_stage) | ID element. Usually used to indicate role, e.g. 'prod', 'staging', 'source', 'build', 'test', 'deploy', 'release' | `string` | `null` | no |
| <a name="input_tags"></a> [tags](#input\_tags) | Additional tags (e.g. `{'BusinessUnit': 'XYZ'}`).<br>Neither the tag keys nor the tag values will be modified by this module. | `map(string)` | `{}` | no |
| <a name="input_tenant"></a> [tenant](#input\_tenant) | ID element \_(Rarely used, not included by default)\_. A customer identifier, indicating who this instance of a resource is for | `string` | `null` | no |
| <a name="input_vault_enabled"></a> [vault\_enabled](#input\_vault\_enabled) | Whether or not a new Vault should be created | `bool` | `true` | no |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_backup_plan_arn"></a> [backup\_plan\_arn](#output\_backup\_plan\_arn) | Backup Plan ARN |
| <a name="output_backup_plan_version"></a> [backup\_plan\_version](#output\_backup\_plan\_version) | Unique, randomly generated, Unicode, UTF-8 encoded string that serves as the version ID of the backup plan |
| <a name="output_backup_selection_id"></a> [backup\_selection\_id](#output\_backup\_selection\_id) | Backup Selection ID |
| <a name="output_backup_vault_arn"></a> [backup\_vault\_arn](#output\_backup\_vault\_arn) | Backup Vault ARN |
| <a name="output_backup_vault_id"></a> [backup\_vault\_id](#output\_backup\_vault\_id) | Backup Vault ID |
<!-- END OF PRE-COMMIT-TERRAFORM DOCS HOOK -->
<!-- prettier-ignore-end -->

## References

- [cloudposse/terraform-aws-components](https://github.com/cloudposse/terraform-aws-components/tree/main/modules/aws-backup) -
  Cloud Posse's upstream component



## Related How-to Guides

- [How to Enable Cross-Region Backups in AWS-Backup](https://docs.cloudposse.com/layers/data/tutorials/how-to-enable-cross-region-backups-in-aws-backup/)


---
> [!NOTE]
> This project is part of Cloud Posse's comprehensive ["SweetOps"](https://cpco.io/homepage?utm_source=github&utm_medium=readme&utm_campaign=cloudposse-terraform-components/aws-backup&utm_content=) approach towards DevOps.
> <details><summary><strong>Learn More</strong></summary>
>
> It's 100% Open Source and licensed under the [APACHE2](LICENSE).
>
> </details>

<a href="https://cloudposse.com/readme/header/link?utm_source=github&utm_medium=readme&utm_campaign=cloudposse-terraform-components/aws-backup&utm_content=readme_header_link"><img src="https://cloudposse.com/readme/header/img"/></a>











## Related Projects

Check out these related projects.

- [Cloud Posse Terraform Modules](https://docs.cloudposse.com/modules/) - Our collection of reusable Terraform modules used by our reference architectures.
- [Atmos](https://atmos.tools) - Atmos is like docker-compose but for your infrastructure

## ✨ Contributing

This project is under active development, and we encourage contributions from our community.
Many thanks to our outstanding contributors:

<a href="https://github.com/cloudposse-terraform-components/aws-backup/graphs/contributors">
  <img src="https://contrib.rocks/image?repo=cloudposse-terraform-components/aws-backup&max=24" />
</a>

### 🐛 Bug Reports & Feature Requests

Please use the [issue tracker](https://github.com/cloudposse-terraform-components/aws-backup/issues) to report any bugs or file feature requests.

### 💻 Developing

If you are interested in being a contributor and want to get involved in developing this project or help out with Cloud Posse's other projects, we would love to hear from you!
Hit us up in [Slack](https://cpco.io/slack?utm_source=github&utm_medium=readme&utm_campaign=cloudposse-terraform-components/aws-backup&utm_content=slack), in the `#cloudposse` channel.

In general, PRs are welcome. We follow the typical "fork-and-pull" Git workflow.
 1. Review our [Code of Conduct](https://github.com/cloudposse-terraform-components/aws-backup/?tab=coc-ov-file#code-of-conduct) and [Contributor Guidelines](https://github.com/cloudposse/.github/blob/main/CONTRIBUTING.md).
 2. **Fork** the repo on GitHub
 3. **Clone** the project to your own machine
 4. **Commit** changes to your own branch
 5. **Push** your work back up to your fork
 6. Submit a **Pull Request** so that we can review your changes

**NOTE:** Be sure to merge the latest changes from "upstream" before making a pull request!

### 🌎 Slack Community

Join our [Open Source Community](https://cpco.io/slack?utm_source=github&utm_medium=readme&utm_campaign=cloudposse-terraform-components/aws-backup&utm_content=slack) on Slack. It's **FREE** for everyone! Our "SweetOps" community is where you get to talk with others who share a similar vision for how to rollout and manage infrastructure. This is the best place to talk shop, ask questions, solicit feedback, and work together as a community to build totally *sweet* infrastructure.

### 📰 Newsletter

Sign up for [our newsletter](https://cpco.io/newsletter?utm_source=github&utm_medium=readme&utm_campaign=cloudposse-terraform-components/aws-backup&utm_content=newsletter) and join 3,000+ DevOps engineers, CTOs, and founders who get insider access to the latest DevOps trends, so you can always stay in the know.
Dropped straight into your Inbox every week — and usually a 5-minute read.

### 📆 Office Hours <a href="https://cloudposse.com/office-hours?utm_source=github&utm_medium=readme&utm_campaign=cloudposse-terraform-components/aws-backup&utm_content=office_hours"><img src="https://img.cloudposse.com/fit-in/200x200/https://cloudposse.com/wp-content/uploads/2019/08/Powered-by-Zoom.png" align="right" /></a>

[Join us every Wednesday via Zoom](https://cloudposse.com/office-hours?utm_source=github&utm_medium=readme&utm_campaign=cloudposse-terraform-components/aws-backup&utm_content=office_hours) for your weekly dose of insider DevOps trends, AWS news and Terraform insights, all sourced from our SweetOps community, plus a _live Q&A_ that you can’t find anywhere else.
It's **FREE** for everyone!

## About

This project is maintained by <a href="https://cpco.io/homepage?utm_source=github&utm_medium=readme&utm_campaign=cloudposse-terraform-components/aws-backup&utm_content=">Cloud Posse, LLC</a>.
<a href="https://cpco.io/homepage?utm_source=github&utm_medium=readme&utm_campaign=cloudposse-terraform-components/aws-backup&utm_content="><img src="https://cloudposse.com/logo-300x69.svg" align="right" /></a>

We are a [**DevOps Accelerator**](https://cpco.io/commercial-support?utm_source=github&utm_medium=readme&utm_campaign=cloudposse-terraform-components/aws-backup&utm_content=commercial_support) for funded startups and enterprises.
Use our ready-to-go terraform architecture blueprints for AWS to get up and running quickly.
We build it with you. You own everything. Your team wins. Plus, we stick around until you succeed.

<a href="https://cpco.io/commercial-support?utm_source=github&utm_medium=readme&utm_campaign=cloudposse-terraform-components/aws-backup&utm_content=commercial_support"><img alt="Learn More" src="https://img.shields.io/badge/learn%20more-success.svg?style=for-the-badge"/></a>

*Your team can operate like a pro today.*

Ensure that your team succeeds by using our proven process and turnkey blueprints. Plus, we stick around until you succeed.

<details>
  <summary>📚 <strong>See What's Included</strong></summary>

- **Reference Architecture.** You'll get everything you need from the ground up built using 100% infrastructure as code.
- **Deployment Strategy.** You'll have a battle-tested deployment strategy using GitHub Actions that's automated and repeatable.
- **Site Reliability Engineering.** You'll have total visibility into your apps and microservices.
- **Security Baseline.** You'll have built-in governance with accountability and audit logs for all changes.
- **GitOps.** You'll be able to operate your infrastructure via Pull Requests.
- **Training.** You'll receive hands-on training so your team can operate what we build.
- **Questions.** You'll have a direct line of communication between our teams via a Shared Slack channel.
- **Troubleshooting.** You'll get help to triage when things aren't working.
- **Code Reviews.** You'll receive constructive feedback on Pull Requests.
- **Bug Fixes.** We'll rapidly work with you to fix any bugs in our projects.
</details>

<a href="https://cloudposse.com/readme/commercial-support/link?utm_source=github&utm_medium=readme&utm_campaign=cloudposse-terraform-components/aws-backup&utm_content=readme_commercial_support_link"><img src="https://cloudposse.com/readme/commercial-support/img"/></a>
## License

<a href="https://opensource.org/licenses/Apache-2.0"><img src="https://img.shields.io/badge/License-Apache%202.0-blue.svg?style=for-the-badge" alt="License"></a>

<details>
<summary>Preamble to the Apache License, Version 2.0</summary>
<br/>
<br/>



```text
Licensed to the Apache Software Foundation (ASF) under one
or more contributor license agreements.  See the NOTICE file
distributed with this work for additional information
regarding copyright ownership.  The ASF licenses this file
to you under the Apache License, Version 2.0 (the
"License"); you may not use this file except in compliance
with the License.  You may obtain a copy of the License at

  https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing,
software distributed under the License is distributed on an
"AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
KIND, either express or implied.  See the License for the
specific language governing permissions and limitations
under the License.
```
</details>

## Trademarks

All other trademarks referenced herein are the property of their respective owners.
---
Copyright © 2017-2024 [Cloud Posse, LLC](https://cpco.io/copyright)


<a href="https://cloudposse.com/readme/footer/link?utm_source=github&utm_medium=readme&utm_campaign=cloudposse-terraform-components/aws-backup&utm_content=readme_footer_link"><img alt="README footer" src="https://cloudposse.com/readme/footer/img"/></a>

<img alt="Beacon" width="0" src="https://ga-beacon.cloudposse.com/UA-76589703-4/cloudposse-terraform-components/aws-backup?pixel&cs=github&cm=readme&an=aws-backup"/>
