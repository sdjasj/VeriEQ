`timescale 1ns/1ps
`define stop $stop
`define checkd(gotv,expv) do if ((gotv) !== (expv)) begin $write("%%Error: %s:%0d:  got=%0d exp=%0d\n", `__FILE__,`__LINE__, (gotv), (expv)); `stop; end while(0);

module out24_logic_simplify (
    input  wire               clock_0,
    input  wire               clock_1,
    input  wire signed [28:28] in1,
    output wire        [21:5] out24
);
    reg signed [21:8] reg_10;

    initial begin
        reg_10 = 0;
    end

    assign out24 = reg_10;

    always @(negedge clock_1 or posedge clock_0) begin
        if (clock_0) begin
            reg_10 <= 0;
        end else begin
            reg_10[14:8] <= {1'b1, ~((in1[28:28] & ~(in1[28:28])))};  // Should assign 3
        end
    end
endmodule

module tb_out24_logic_print;

    reg  clock_0;
    reg  clock_1;
    reg  signed [28:28] in1;
    wire [21:5] out24;
    bit cmp;

    out24_logic_simplify dut (
        .clock_0 (clock_0),
        .clock_1 (clock_1),
        .in1     (in1),
        .out24   (out24)
    );

    initial begin
        clock_0 = 0;
        clock_1 = 1;
        in1     = 1'b0;
        cmp = out24 == 0;
        `checkd(cmp, 1);
        #2;
        clock_1 = 0;
        #2;
        cmp = out24 == 1; // line 54
        `checkd(cmp, 1);
        cmp = out24 == 3;
        `checkd(cmp, 1); // line 57
        $finish;
    end
endmodule