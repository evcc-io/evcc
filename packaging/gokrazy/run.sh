#!/usr/bin/env bash

imagefile="${IMAGE_FILE:=drive.img}"
arch="${MACHINE:=amd64}"
os=""

if [ "$(uname)" == "Darwin" ]; then
    os="darwin"
elif [ "$(expr substr $(uname -s) 1 5)" == "Linux" ]; then
    os="linux"
fi


case $arch in
    # arm64 emulator with kernel auto loading
    arm64)
        args=(
            -name gokrazy-arm64
            -m 3G
            -smp 2
            -M virt,highmem=off
            -cpu cortex-a72
            -nographic
            -drive file=${imagefile},format=raw
            -bios ./packaging/gokrazy/QEMU_EFI.fd
        )

        if [[ "$os" == "darwin" ]]; then
            args+=(-nic vmnet-shared)
        else
            args+=(-netdev user,id=net0,hostfwd=tcp::8080-:80,hostfwd=tcp::2222-:22)
            args+=(-device e1000,netdev=net0)
        fi
        qemu-system-aarch64 "${args[@]}"
        ;;

    # raspi3b emulator with manual kernel loading
    # requires kernel (./vmlinuz) and dtb file extraction from the drive.img,
    # to be used as an arg (with -dtb, -kernel and -append)
    # to instruct the vm on how to load the kernel
    raspi3b)
        if [[ "$os" == "darwin" ]]; then
            args+=(-nic vmnet-shared)
        fi

        # Only works with qemu === v5.2.0
        # because for lower versions networking is missing.
        # It was introduced in qemu v5.2.0 via usb networking (-device usb-net), but it is very very slow at the point that gokrazy network updates fail.
        # For qemu versions >= v6.0.0, the /gokrazy/init process crashes early at gokrazy.Boot(). To be investigated.
        qemu_version="$(qemu-system-aarch64 --version | sed -nr 's/^.*version\s([.0-9]*).*$/\1/p')"
        if [[ "$qemu_version" != "5.2.0" ]]; then
			echo "error: incompatible qemu-system-aarch64 version: $qemu_version. gokrazy on raspi3b can only run on 5.2.0"
			exit 1;
        fi

        # Extract the kernel (vmlinuz) and the dtb file from the drive.img first
        ./extract_kernel.sh
        args=(
            -name gokrazy-arm64-raspi3b
            -m 1024
            -no-reboot
            -M raspi3b
            -append "console=tty1 console=ttyAMA0,115200 dwc_otg.fiq_fsm_enable=0 root=/dev/mmcblk0p2 init=/gokrazy/init rootwait panic=10 oops=panic"
            -dtb ./bcm2710-rpi-3-b-plus.dtb
            -nographic
            -serial mon:stdio
            -drive file=${imagefile},format=raw
            -kernel vmlinuz
            -netdev user,id=net0,hostfwd=tcp::8080-:80,hostfwd=tcp::2222-:22
            -device usb-net,netdev=net0
        )

        qemu-system-aarch64 "${args[@]}"
        ;;

    # amd64 emulator with kernel auto loading
    amd64)
        args=(
            -name gokrazy-amd64
            -m 3G
            -smp $(nproc)
            -usb
            -nographic
            -serial mon:stdio
            -boot order=d
            -drive file=${imagefile},format=raw
        )

        if [[ "$os" == "darwin" ]]; then
            args+=(-nic vmnet-shared)
        else
            args+=(-netdev user,id=net0,hostfwd=tcp::8080-:80,hostfwd=tcp::2222-:22)
            args+=(-device e1000,netdev=net0)
        fi

        qemu-system-x86_64 "${args[@]}"
        ;;

    *)
        echo -n "unsupported arch ${arch}"
        exit
        ;;
esac
