+++
date = '2025-09-21T21:15:47+02:00'
title = "Introduction"
linkTitle = "Introduction"
description = "Introduction to token-operator"
weight = 10
[params]
  menuPre = "<i class='fa-fw fas fa-star'></i> "
+++

## Introduction to token-operator

When you deal with a GitLab instance that has many groups, sub-groups and users, you may run into the situation
that you're missing an overview of what access tokens exist.

Additionally, you may want to automate the rotation of existing GitLab access tokens, so that they have a short life-span
and can get rotated on-demand when needed while not wasting precious time of your staff.

If you provide access to GitLab access tokens through a vault instance, you can additionally monitor and control access
to the access tokens.

To solve these issues, 'token-operator' was born.

### What token-operator does

1. It connects to the configured GitLab and password vault.
2. Loops over the tokens in the configuration.
3. For each token, decides if it needs rotating and if so, rotates the token and updates the vault item.


Continue to 👉 [Initial setup](./initial-setup.md)