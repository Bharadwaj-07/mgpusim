import re
import subprocess
import argparse
import os
import sys
sys.path.append(os.getcwd() + '/../common/')
from benchmarking import *


import pandas as pd

data_columns = ['benchmark', 'env', 'num_access', 'time']

def generate_benchmark(count):
    with open('microbench/kernels_template.asm', 'r') as template_file:
        template = template_file.read()

    pre_scan_insts = ''
    for i in range(0, 16384):
        pre_scan_insts += 'flat_load_dword v0, v[4:5]\n'
        pre_scan_insts += 'v_add_u32 v4, vcc, v4, v3\n'
        pre_scan_insts += 'v_addc_u32 v5, vcc, v5, 0, vcc\n'
        pre_scan_insts += 's_waitcnt vmcnt(0)\n'

    insts = ''
    for i in range(0, count):
        insts += 'flat_load_dword v0, v[1:2]\n'
        insts += 'v_add_u32 v1, vcc, v1, v3\n'
        insts += 'v_addc_u32 v2, vcc, v2, 0, vcc\n'
        insts += 's_waitcnt vmcnt(0)\n'
    kernel = template.format(pre_scan_insts, insts)

    with open('microbench/kernels.asm', 'w') as kernel_file:
        kernel_file.write(kernel)

    p = subprocess.Popen('make kernels.hsaco', cwd='microbench',
                         shell=True)
    p.wait()


def run_on_simulator(num_access):
    """ run benchmark and retuns a data frame that represents its result """
    data = pd.DataFrame(columns=data_columns)

    process = subprocess.Popen("go build", shell=True, cwd='.',
                            stdout=subprocess.DEVNULL)
    process.wait()

    duration = run_benchmark_on_simulator('./l2_read -timing', os.getcwd())
    entry = ['l2_read', 'sim', num_access , duration]
    print(entry)
    data = data.append(
        pd.DataFrame([entry], columns=data_columns),
        ignore_index=True,
    )

    return data


def run_on_gpu(num_access, repeat):
    data = pd.DataFrame(columns=data_columns)

    for i in range(0, repeat):
        duration = run_benchmark_on_gpu(
            './kernel', os.getcwd() + '/microbench/')
        entry = ['dram_read', 'gpu', num_access, duration]
        print(entry)
        data = data.append(
            pd.DataFrame([entry], columns=data_columns),
            ignore_index=True,
        )

    return data


def parse_args():
    parser = argparse.ArgumentParser(description='L1_Read microbenchmark')
    parser.add_argument('--gpu', dest='gpu', action='store_true')
    parser.add_argument('--sim', dest='sim', action='store_true')
    parser.add_argument('--repeat', type=int, default=20)
    args = parser.parse_args()
    return args


def main():
    args = parse_args()

    num_access_list = range(0, 129, 1)

    if args.gpu:
        data = pd.DataFrame(columns=data_columns)
        for num_access in num_access_list:
            generate_benchmark(num_access)
            results = run_on_gpu(num_access, args.repeat)
            data = data.append(results, ignore_index=True)
        data.to_csv('gpu.csv')

    if args.sim:
        data = pd.DataFrame(columns=data_columns)
        for num_access in num_access_list:
            generate_benchmark(num_access)
            results = run_on_simulator(num_access)
            data = data.append(results, ignore_index=True)
        data.to_csv('sim.csv')


if __name__ == '__main__':
    main()

