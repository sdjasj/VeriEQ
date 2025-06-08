`timescale 1ns/1ps

module tb_dut_module;

    parameter NUM_VECTORS = 100;  // 你想读取的行数



    reg  [31:0] in0,  in1,  in2,  in3,  in4,
                in5,  in6,  in7,  in8,  in9,
                in10, in11, in12, in13, in14,
                in15, in16, in17, in18, in19;

    wire [31:0] out0, out1,  out2,  out3,  out4,
                out5, out6,  out7,  out8,  out9,
                out10,out11, out12, out13, out14,
                out15,out16, out17, out18, out19;

    dut_module uut (
        .in0(in0),   .in1(in1),   .in2(in2),   .in3(in3),   .in4(in4),
        .in5(in5),   .in6(in6),   .in7(in7),   .in8(in8),   .in9(in9),
        .in10(in10), .in11(in11), .in12(in12), .in13(in13), .in14(in14),
        .in15(in15), .in16(in16), .in17(in17), .in18(in18), .in19(in19),

        .out0(out0),   .out1(out1),   .out2(out2),   .out3(out3),   .out4(out4),
        .out5(out5),   .out6(out6),   .out7(out7),   .out8(out8),   .out9(out9),
        .out10(out10), .out11(out11), .out12(out12), .out13(out13), .out14(out14),
        .out15(out15), .out16(out16), .out17(out17), .out18(out18), .out19(out19)
    );

    // ---------------------------
    // 4) 声明文件句柄、辅助变量
    // ---------------------------
    integer fin, fout;
    integer i, status;
    reg [31:0] output_hash;

    // ---------------------------
    // 5) 主测试过程
    // ---------------------------
    initial begin
        // 尝试打开输入文件
        fin = $fopen("test_input.txt", "r");
        if (fin == 0) begin
            $display("ERROR: Cannot open test_input.txt");
            $finish;
        end

        // 打开输出文件
        fout = $fopen("test_output.txt", "w");
        if (fout == 0) begin
            $display("ERROR: Cannot open test_output.txt for writing");
            $finish;
        end

        // 初始化输入
        in0  = 0;  in1  = 0;  in2  = 0;  in3  = 0;  in4  = 0;
        in5  = 0;  in6  = 0;  in7  = 0;  in8  = 0;  in9  = 0;
        in10 = 0;  in11 = 0;  in12 = 0;  in13 = 0;  in14 = 0;
        in15 = 0;  in16 = 0;  in17 = 0;  in18 = 0;  in19 = 0;
        output_hash = 32'h0;

        // --------------------------------------
        //  只循环指定次数 (NUM_VECTORS)，读取文件
        // --------------------------------------
        for (i = 0; i < NUM_VECTORS; i = i + 1) begin
            // 尝试扫描 20 个数据
            status = $fscanf(fin,
                "%h %h %h %h %h %h %h %h %h %h "
                "%h %h %h %h %h %h %h %h %h %h\n",
                in0,  in1,  in2,  in3,  in4,
                in5,  in6,  in7,  in8,  in9,
                in10, in11, in12, in13, in14,
                in15, in16, in17, in18, in19
            );

            // 检查是否实际读到 20 个数
            if (status < 20) begin
                $display("WARNING: File doesn't have enough lines or format error at line %0d", i);
                // 可以选择 break 或者直接 finish
                // 这里选择结束仿真
                $finish;
            end

            // 等待一段时间(给电路反应)
            #10;

            // (可选) 计算输出 hash - 简单做 XOR
            output_hash = 32'hABCD_1234
                        ^ out0  ^ out1  ^ out2  ^ out3  ^ out4
                        ^ out5  ^ out6  ^ out7  ^ out8  ^ out9
                        ^ out10 ^ out11 ^ out12 ^ out13 ^ out14
                        ^ out15 ^ out16 ^ out17 ^ out18 ^ out19;

            // 将结果写到输出文件
            $fwrite(fout, "Vector %0d: out0=%h out1=%h ... out19=%h  HASH=%h\n",
                            i,
                            out0, out1, /* ... */ out19,
                            output_hash
            );
        end

        // 循环结束后，关闭文件并结束
        $fclose(fin);
        $fclose(fout);
        $display("Simulation finished after %0d vectors.", NUM_VECTORS);
        $finish;
    end

endmodule
