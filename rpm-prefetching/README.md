In order for our builds to be hermetic (without network access) we configure rpm-prefetching.

If you want to prefetch more RPMs in order to install them during the build, you need to update the `rpms.in.yaml` file and follow [this doc](https://konflux.pages.redhat.com/docs/users/how-tos/configuring/activation-keys-subscription.html#_configuring_an_rpm_lockfile_for_hermetic_builds) to generate the `rpms.lock.yaml` file.
There are a few small things you should do differently from the guide:
1. You do not need to create a new activation-key, instead use [this one](https://console.redhat.com/insights/connector/activation-keys/assisted-installer).
2. The image you should run is the base image we use in the dockerfile.
3. When trying to run the `subscription-manager` command you might get the following error: `subscription-manager is disabled when running inside a container. Please refer to your host system for subscription management.` and the fix is to run `rm /etc/rhsm-host`, this will remove a symlink and then you can rerun the `subscription-manager` command.
4. You shouldn't manually update the `redhat.repo` file. after running the previous commands in the guide, it should be fine.
5. For the `rpm-lockfile-prototype` command, use `rpm-lockfile-prototype --image <base image> rpms.in.yaml` for simplicity.

If you want to better understand the `rpms.in.yaml` file you can look at the project's README [here](https://github.com/konflux-ci/rpm-lockfile-prototype/blob/main/README.md).
