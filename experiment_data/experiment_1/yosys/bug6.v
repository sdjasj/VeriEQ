module a(
    input f,
    input [1:0] b,
    output reg c
);
    reg [1:0] g, h, i, j;
    wire k, e;

    always @(*) begin
        begin
            c = h;
        end

        case (b)
            'd1: begin
                h = i;
            end
        endcase
    end

    always @(f) begin
        case (g << j >> h)
            32'd3: begin
                case (b << j)
                    'd1: begin
                        h = 0;
                        j = f ? (^k) : 0;
                    end
                    'd2: begin
                        i = j;
                    end
                endcase
            end

            default: begin
                h = e || b;
            end
        endcase
    end
endmodule