#define CL_KERNEL_LOOP(i, n)                        \
  for (int i = get_group_id(0) * get_local_size(0) + get_local_id(0); \
      i < (n);                                       \
      i += get_local_size(0) * get_num_groups(0))

__kernel void MaxPoolForward(const int nthreads, __global float* bottom_data, const int num, const int channels, const int height, const int width, const int pooled_height, const int pooled_width, const int kernel_h, const int kernel_w, const int stride_h, const int stride_w, const int pad_h, const int pad_w, __global float* top_data, __global int* mask_data, __global int* indexs) {
  CL_KERNEL_LOOP(index, num * channels * pooled_height * pooled_width) {
    indexs[index] = index;
    indexs[3] = get_local_size(0) * get_num_groups(0);
    int pw = index % pooled_width;
    int ph = (index / pooled_width) % pooled_height;
    int c = (index / pooled_width / pooled_height) % channels;
    int n = index / pooled_width / pooled_height / channels;
    int hstart = ph * stride_h - pad_h;
    int wstart = pw * stride_w - pad_w;
    int ppw = pooled_height +10;
    int pph = pooled_width +10;
    int ch = channels + 10;
    int preph = index / pooled_width;
    int aftph = preph % pooled_height;
    int test = 0 % 1;
    indexs[20] = ppw;
    indexs[21] = pph;
    indexs[22] = ch;
    indexs[23] = preph;
    indexs[24] = aftph;
    indexs[25] = test;
    indexs[5+5*index] = pw;
    indexs[6+5*index] = ph;
    indexs[7+5*index] = c;
    indexs[8+5*index] = n;
    const int hend = min(hstart + kernel_h, height);
    const int wend = min(wstart + kernel_w, width);
    hstart = max(hstart, 0);
    wstart = max(wstart, 0);
    float maxval = -1;
    int maxidx = -1;
    bottom_data = bottom_data + (n * channels + c) * height * width;
    for (int h = hstart; h < hend; ++h) {
      for (int w = wstart; w < wend; ++w) {
        top_data[0] = bottom_data[h * width + w];
        if (bottom_data[h * width + w] > maxval) {
          maxidx = h * width + w;
          maxval = bottom_data[maxidx];
        }
      }
    }
    //top_data[index] = maxval;
    mask_data[index] = maxidx;
  }
}

__kernel void MaxPoolBackward(
    const int nthreads,
    __global float* top_diff,
    __global int* top_mask,
    const int num, const int channels,
    const int height, const int width, const int pooled_height,
    const int pooled_width, const int kernel_h, const int kernel_w,const int stride_h, const int stride_w, const int pad_h, const int pad_w,__global float* bottom_diff
  ) {

  int gid = get_global_id(0);
  int tmp = get_global_size(0);
  for(int index = gid; index < nthreads; index += tmp) {
    int w = index % width;
    int h = (index / width) % height;
    int c = (index / width / height) % channels;
    int n = index / width / height / channels;
    int phstart =
        (h + pad_h < kernel_h) ? 0 : (h + pad_h - kernel_h) / stride_h + 1;
    int phend = min((h + pad_h) / stride_h + 1, pooled_height);
    int pwstart =
        (w + pad_w < kernel_w) ? 0 : (w + pad_w - kernel_w) / stride_w + 1;
    int pwend = min((w + pad_w) / stride_w + 1, pooled_width);
    float gradient = 0;
    int offset = (n * channels + c) * pooled_height * pooled_width;
    top_diff += offset;
    top_mask += offset;
    for (int ph = phstart; ph < phend; ++ph) {
      for (int pw = pwstart; pw < pwend; ++pw) {
          if (top_mask[ph * pooled_width + pw] - 1 == h * width + w) {
              gradient += top_diff[ph * pooled_width + pw];
          }
      }
    }
    bottom_diff[index] = gradient;
  }
}
