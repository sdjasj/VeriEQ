`timescale 1ns/1ps

module tb_dut_module;

    parameter NUM_VECTORS = 20;  // 你想读取的行数

reg in0;
    reg [28:18] in1;
    reg [31:15] in2;
    reg [1:0] in3;
reg in4;
    reg [30:24] in5;
    reg [19:9] in6;
    reg [11:8] in7;
    reg [10:2] in8;
    reg [10:8] in9;
    reg [19:7] in10;
    reg [21:15] in11;
    reg [18:14] in12;
    reg [12:5] in13;
    reg [24:13] in14;
reg in15;
    reg [12:4] in16;
    reg [20:10] in17;
    reg [35:15] in18;
    reg [10:5] in19;
reg clock_0;
reg clock_1;
reg clock_2;
reg clock_3;
reg clock_4;
reg clock_5;
reg clock_6;
reg clock_7;
reg clock_8;
reg clock_9;
reg clock_10;
reg clock_11;
reg clock_12;
reg clock_13;
reg clock_14;
reg clock_15;
reg clock_16;
reg clock_17;
reg clock_18;
reg clock_19;
wire out0;
wire out1;
    wire [11:3] out2;
    wire [8:1] out3;
    wire [23:9] out4;
    wire [32:20] out5;
    wire [0:0] out6;
    wire [4:4] out7;
    wire [34:8] out8;
    wire [35:16] out9;
    wire [30:18] out10;
wire out11;
wire out12;
    wire [17:0] out13;
    wire [19:8] out14;
    wire [7:6] out15;
    wire [17:13] out16;
    wire [10:0] out17;
wire out18;
wire out19;

    itmozsmldb uut  (
		.in0(in0), .in1(in1), .in2(in2), .in3(in3), .in4(in4), .in5(in5), .in6(in6), .in7(in7), .in8(in8), .in9(in9), .in10(in10), .in11(in11), .in12(in12), .in13(in13), .in14(in14), .in15(in15), .in16(in16), .in17(in17), .in18(in18), .in19(in19), .clock_0(clock_0), .clock_1(clock_1), .clock_2(clock_2), .clock_3(clock_3), .clock_4(clock_4), .clock_5(clock_5), .clock_6(clock_6), .clock_7(clock_7), .clock_8(clock_8), .clock_9(clock_9), .clock_10(clock_10), .clock_11(clock_11), .clock_12(clock_12), .clock_13(clock_13), .clock_14(clock_14), .clock_15(clock_15), .clock_16(clock_16), .clock_17(clock_17), .clock_18(clock_18), .clock_19(clock_19), 
		.out0(out0), .out1(out1), .out2(out2), .out3(out3), .out4(out4), .out5(out5), .out6(out6), .out7(out7), .out8(out8), .out9(out9), .out10(out10), .out11(out11), .out12(out12), .out13(out13), .out14(out14), .out15(out15), .out16(out16), .out17(out17), .out18(out18), .out19(out19)
    );
    integer fin, fout;
    integer i, status;
    reg [31:0] output_hash;
    initial begin
        // 尝试打开输入文件
        fin = $fopen("input.txt", "r");
        if (fin == 0) begin
            $display("ERROR: Cannot open input.txt");
            $finish;
        end
		
        // 打开输出文件
        fout = $fopen("output.txt", "w");
        if (fout == 0) begin
            $display("ERROR: Cannot open output.txt for writing");
            $finish;
        end
        // 初始化输入
		in0 = 0; in1 = 0; in2 = 0; in3 = 0; in4 = 0; in5 = 0; in6 = 0; in7 = 0; in8 = 0; in9 = 0; in10 = 0; in11 = 0; in12 = 0; in13 = 0; in14 = 0; in15 = 0; in16 = 0; in17 = 0; in18 = 0; in19 = 0; clock_0 = 0; clock_1 = 0; clock_2 = 0; clock_3 = 0; clock_4 = 0; clock_5 = 0; clock_6 = 0; clock_7 = 0; clock_8 = 0; clock_9 = 0; clock_10 = 0; clock_11 = 0; clock_12 = 0; clock_13 = 0; clock_14 = 0; clock_15 = 0; clock_16 = 0; clock_17 = 0; clock_18 = 0; clock_19 = 0; 
        output_hash = 32'h0;
		
        // --------------------------------------
        //  只循环指定次数 (NUM_VECTORS)，读取文件
        // --------------------------------------
		#2000;
        for (i = 0; i < NUM_VECTORS; i = i + 1) begin
            status = $fscanf(fin, "%d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d\n", in0 ,in1 ,in2 ,in3 ,in4 ,in5 ,in6 ,in7 ,in8 ,in9 ,in10 ,in11 ,in12 ,in13 ,in14 ,in15 ,in16 ,in17 ,in18 ,in19 ,clock_0 ,clock_1 ,clock_2 ,clock_3 ,clock_4 ,clock_5 ,clock_6 ,clock_7 ,clock_8 ,clock_9 ,clock_10 ,clock_11 ,clock_12 ,clock_13 ,clock_14 ,clock_15 ,clock_16 ,clock_17 ,clock_18 ,clock_19 );

            // 检查是否实际读到 20 个数
            if (status < 40) begin
                $display("WARNING: File doesn't have enough lines or format error at line %0d", i);
                // 可以选择 break 或者直接 finish
                // 这里选择结束仿真
                $finish;
            end

            // 等待一段时间(给电路反应)
            #2000;
			$fwrite(fout, "%0d %0d %0d %0d %0d %0d %0d %0d %0d %0d %0d %0d %0d %0d %0d %0d %0d %0d %0d %0d\n", out0, out1, out2, out3, out4, out5, out6, out7, out8, out9, out10, out11, out12, out13, out14, out15, out16, out17, out18, out19);
        end

        // 循环结束后，关闭文件并结束
        $fclose(fin);
        $fclose(fout);
        $display("Simulation finished after %0d vectors.", NUM_VECTORS);
        $finish;
    end

endmodule
